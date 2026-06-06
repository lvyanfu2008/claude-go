package app

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/messagerow"
	"goc/types"
	state "goc/gou/app/state"
)

func (m *model) clearTranscriptSearchState() {
	m.Screen.SearchOpen = false
	m.Screen.SearchQuery = ""
	m.Screen.SearchHits = nil
	m.Screen.SearchCursor = 0
}

// plainMessageSearchText builds a lowercased haystack for transcript substring search (TS transcript / search).
func plainMessageSearchText(msg types.Message) string {
	msg = messagerow.NormalizeMessageJSON(msg)
	var b strings.Builder
	b.WriteString(strings.ToLower(string(msg.Type)))
	b.WriteByte(' ')
	switch msg.Type {
	case types.MessageTypeCollapsedReadSearch:
		b.WriteString(strings.ToLower(messagerow.SearchReadSummaryTextFromMessage(false, msg)))
		for _, p := range msg.ReadFilePaths {
			b.WriteByte(' ')
			b.WriteString(strings.ToLower(p))
		}
		for _, a := range msg.SearchArgs {
			b.WriteByte(' ')
			b.WriteString(strings.ToLower(a))
		}
		if msg.DisplayMessage != nil {
			b.WriteByte(' ')
			b.WriteString(plainMessageSearchText(*msg.DisplayMessage))
		}
		return b.String()
	case types.MessageTypeGroupedToolUse:
		b.WriteString(strings.ToLower(msg.ToolName))
		for i := range msg.Messages {
			b.WriteByte(' ')
			b.WriteString(plainMessageSearchText(msg.Messages[i]))
		}
		for i := range msg.Results {
			b.WriteByte(' ')
			b.WriteString(plainMessageSearchText(msg.Results[i]))
		}
		return b.String()
	default:
		if len(msg.Content) == 0 {
			return b.String()
		}
		var blocks []types.MessageContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			b.WriteString(strings.ToLower(string(msg.Content)))
			return b.String()
		}
		for _, bl := range blocks {
			switch bl.Type {
			case "text":
				b.WriteString(strings.ToLower(bl.Text))
				b.WriteByte(' ')
			case "tool_use", "server_tool_use":
				b.WriteString(strings.ToLower(bl.Name))
				b.WriteByte(' ')
			}
		}
		return b.String()
	}
}

func plainTranscriptStreamingToolSearchText(group GroupedStreamingTool) string {
	if !group.IsGroup {
		tu := group.Single
		return strings.ToLower(strings.TrimSpace(tu.Name) + " " + strings.TrimSpace(tu.ToolUseID) + " " + tu.UnparsedInput)
	}
	var b strings.Builder
	for _, tu := range group.Items {
		b.WriteString(strings.TrimSpace(tu.Name))
		b.WriteByte(' ')
		b.WriteString(strings.TrimSpace(tu.ToolUseID))
		b.WriteByte(' ')
		b.WriteString(tu.UnparsedInput)
		b.WriteByte(' ')
	}
	return strings.ToLower(b.String())
}

func transcriptSearchHLStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color("58")).Foreground(lipgloss.Color("230"))
}

// highlightSearchPlain wraps case-insensitive needle matches in hl (terminal TS useSearchHighlight parity).
func highlightSearchPlain(s, needle string, hl lipgloss.Style) string {
	needle = strings.TrimSpace(needle)
	if needle == "" || s == "" {
		return s
	}
	lowS := strings.ToLower(s)
	lowN := strings.ToLower(needle)
	if lowN == "" {
		return s
	}
	var b strings.Builder
	cur := 0
	for cur < len(s) {
		rel := strings.Index(lowS[cur:], lowN)
		if rel < 0 {
			b.WriteString(s[cur:])
			break
		}
		idx := cur + rel
		b.WriteString(s[cur:idx])
		end := idx + len(lowN)
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(hl.Render(s[idx:end]))
		cur = end
	}
	return b.String()
}

func (m *model) transcriptSearchHighlightNeedle() string {
	if m.Screen.Mode != state.ScreenTranscript || m.Screen.DumpMode {
		return ""
	}
	q := strings.TrimSpace(m.Screen.SearchQuery)
	if q == "" {
		return ""
	}
	return q
}

func (m *model) rebuildTranscriptSearchMatches() {
	msgView := m.messagesForScroll()
	st := m.transcriptStreamingToolsForView()
	rowN := len(msgView) + len(st)
	q := strings.TrimSpace(m.Screen.SearchQuery)
	if q == "" {
		m.Screen.SearchHits = nil
		m.Screen.SearchCursor = 0
		m.rebuildHeightCache()
		return
	}
	needle := strings.ToLower(q)
	var hits []int
	for i := 0; i < rowN; i++ {
		var hay string
		if i < len(msgView) {
			hay = plainMessageSearchText(msgView[i])
		} else {
			hay = plainTranscriptStreamingToolSearchText(st[i-len(msgView)])
		}
		if strings.Contains(hay, needle) {
			hits = append(hits, i)
		}
	}
	m.Screen.SearchHits = hits
	if len(hits) == 0 {
		m.Screen.SearchCursor = 0
		m.rebuildHeightCache()
		return
	}
	if m.Screen.SearchCursor >= len(hits) {
		m.Screen.SearchCursor = 0
	}
	m.rebuildHeightCache()
	m.scrollTranscriptToMessageIndex(hits[m.Screen.SearchCursor])
}

func (m *model) scrollTranscriptToMessageIndex(msgIdx int) {
	keys := m.scrollItemKeys()
	if msgIdx < 0 || msgIdx >= len(keys) {
		return
	}
	off := 0
	for i := 0; i < msgIdx; i++ {
		off += m.Scroll.HeightCache[keys[i]]
	}
	m.Scroll.Top = off
	m.Scroll.Sticky = false
}

func (m *model) transcriptSearchStep(delta int) {
	h := m.Screen.SearchHits
	if len(h) == 0 {
		return
	}
	m.Screen.SearchCursor = (m.Screen.SearchCursor + delta + len(h)) % len(h)
	m.scrollTranscriptToMessageIndex(h[m.Screen.SearchCursor])
}

func (m *model) handleTranscriptSearchBarKey(msg tea.KeyPressMsg) bool {
	if !m.Screen.SearchOpen {
		return false
	}
	s := msg.String()
	switch s {
	case "esc":
		m.clearTranscriptSearchState()
		m.rebuildHeightCache()
		return true
	case "enter":
		m.Screen.SearchOpen = false
		return true
	case "backspace", "ctrl+h":
		if m.Screen.SearchQuery != "" {
			r := []rune(m.Screen.SearchQuery)
			if len(r) > 0 {
				m.Screen.SearchQuery = string(r[:len(r)-1])
				m.rebuildTranscriptSearchMatches()
			}
		}
		return true
	}
	if msg.Key().Text != "" {
		m.Screen.SearchQuery += msg.Key().Text
		m.rebuildTranscriptSearchMatches()
		return true
	}
	return false
}

// handleTranscriptKey returns (handled, cmd). cmd may be non-nil when bracket dump prints to scrollback (TS).
func (m *model) handleTranscriptKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.Screen.Mode != state.ScreenTranscript {
		return false, nil
	}
	if m.handleTranscriptSearchBarKey(msg) {
		return true, nil
	}
	if !m.Screen.SearchOpen && !m.Screen.DumpMode {
		if msg.String() == "/" {
			m.Screen.SearchOpen = true
			m.Screen.SearchQuery = ""
			m.rebuildTranscriptSearchMatches()
			return true, nil
		}
		if strings.TrimSpace(m.Screen.SearchQuery) != "" {
			switch msg.String() {
			case "n":
				m.transcriptSearchStep(1)
				return true, nil
			case "N":
				m.transcriptSearchStep(-1)
				return true, nil
			}
		}
	}
	if !m.Screen.SearchOpen && !m.Screen.DumpMode {
		if m.transcriptScrollByKeyType(msg) {
			return true, nil
		}
	}
	s := msg.String()
	if msg.String() == "ctrl+e" {
		if m.Screen.DumpMode {
			return true, nil
		}
		m.Screen.ShowAll = !m.Screen.ShowAll
		m.rebuildHeightCache()
		return true, nil
	}
	switch s {
	case "ctrl+l":
		return true, teaGlobalRedrawCmd()
	case "ctrl+o":
		return true, m.exitTranscriptScreenWithPostCmd()
	case "esc", "q", "ctrl+c":
		return true, m.exitTranscriptScreenWithPostCmd()
	case "[":
		if m.Screen.DumpMode || m.Screen.SearchOpen {
			return true, nil
		}
		m.Screen.DumpMode = true
		m.Screen.ShowAll = true
		m.rebuildHeightCache()
		plain := transcriptExportPlain(m, exportTranscriptWidth(m))
		m.Screen.SuspendAltScreenForScrollbackDump = gouDemoAltScreenEnabled()
		return true, transcriptBracketDumpScrollbackCmd(plain)
	case "v":
		if m.Screen.SearchOpen {
			return true, nil
		}
		if m.Screen.EditorBusy {
			return true, nil
		}
		gen := m.Screen.EditorGen
		m.Screen.EditorBusy = true
		m.Screen.EditorStatus = fmt.Sprintf("rendering %d messages…", m.transcriptEffectiveN())
		return true, m.transcriptEditorPrepCmd(gen)
	}
	if m.Screen.DumpMode {
		return true, nil
	}
	// TS ScrollKeybindingHandler: isActive && isModal with isModal={!searchOpen} in REPL transcript.
	// Pager keys (arrows, space, j/k, …) do not run while the search bar is open.
	if !m.Screen.SearchOpen {
		switch s {
		case "up":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-1)
			m.transcriptAfterManualScroll()
			return true, nil
		case "down":
			m.Scroll.Sticky = false
			m.Scroll.Top += 1
			m.transcriptAfterManualScroll()
			return true, nil
		case "pgup":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
			m.transcriptAfterManualScroll()
			return true, nil
		case "pgdown":
			m.Scroll.Sticky = false
			m.Scroll.Top += listViewportH(m) / 2
			m.transcriptAfterManualScroll()
			return true, nil
		case "home", "ctrl+home":
			m.Scroll.Sticky = false
			m.Scroll.Top = 0
			m.transcriptAfterManualScroll()
			return true, nil
		case "end", "ctrl+end":
			m.Scroll.Sticky = true
			m.Scroll.Top = 1 << 30
			return true, nil
		// TS modalPagerAction (ScrollKeybindingHandler.tsx): j/k/g/G, ctrl+u/d/b/f, bare b, space, ctrl+n/p.
		case "j":
			m.Scroll.Sticky = false
			m.Scroll.Top += 1
			m.transcriptAfterManualScroll()
			return true, nil
		case "k":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-1)
			m.transcriptAfterManualScroll()
			return true, nil
		case "g":
			m.Scroll.Sticky = false
			m.Scroll.Top = 0
			m.transcriptAfterManualScroll()
			return true, nil
		case "G", "shift+g":
			m.Scroll.Sticky = true
			m.Scroll.Top = 1 << 30
			return true, nil
		case "ctrl+u":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
			m.transcriptAfterManualScroll()
			return true, nil
		case "ctrl+d":
			m.Scroll.Sticky = false
			m.Scroll.Top += listViewportH(m) / 2
			m.transcriptAfterManualScroll()
			return true, nil
		case "ctrl+b":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m))
			m.transcriptAfterManualScroll()
			return true, nil
		case "ctrl+f":
			m.Scroll.Sticky = false
			m.Scroll.Top += listViewportH(m)
			m.transcriptAfterManualScroll()
			return true, nil
		case "b":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m))
			m.transcriptAfterManualScroll()
			return true, nil
		case "space":
			m.Scroll.Sticky = false
			m.Scroll.Top += listViewportH(m)
			m.transcriptAfterManualScroll()
			return true, nil
		case "ctrl+n":
			m.Scroll.Sticky = false
			m.Scroll.Top += 1
			m.transcriptAfterManualScroll()
			return true, nil
		case "ctrl+p":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-1)
			m.transcriptAfterManualScroll()
			return true, nil
		default:
			return true, nil
		}
	}
	return true, nil
}

// transcriptAfterManualScroll pins scrollTop after leaving sticky-bottom (sentinel ~1<<30) so virtualscroll
// stays valid. We only clamp when scrollTop is still in the huge range: without heightCache (e.g. tests),
// full clamp can pin to 0 and break small scrollTop values; the next View also clamps when !sticky.
func (m *model) transcriptAfterManualScroll() {
	if m.Scroll.Sticky {
		return
	}
	if m.Scroll.Top < 1<<20 {
		return
	}
	m.clampScrollTopForVirtualList()
}

// transcriptScrollByKeyType handles Kitty / disambiguated keys where msg.String() is not "up"/"down" (see handleTranscriptKey string switch).
func (m *model) transcriptScrollByKeyType(msg tea.KeyPressMsg) bool {
	switch msg.Key().Code {
	case tea.KeyUp:
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-1)
		m.transcriptAfterManualScroll()
		return true
	case tea.KeyDown:
		m.Scroll.Sticky = false
		m.Scroll.Top += 1
		m.transcriptAfterManualScroll()
		return true
	case tea.KeyPgUp:
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
		m.transcriptAfterManualScroll()
		return true
	case tea.KeyPgDown:
		m.Scroll.Sticky = false
		m.Scroll.Top += listViewportH(m) / 2
		m.transcriptAfterManualScroll()
		return true
	case tea.KeyHome:
		m.Scroll.Sticky = false
		m.Scroll.Top = 0
		m.transcriptAfterManualScroll()
		return true
	case tea.KeyEnd:
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
		return true
	default:
		return false
	}
}

func transcriptSearchStatusLines(m *model) []string {
	if m.Screen.Mode != state.ScreenTranscript {
		return nil
	}
	if m.Screen.SearchOpen {
		q := m.Screen.SearchQuery
		if len(q) > 60 {
			q = q[:57] + "…"
		}
		return []string{fmt.Sprintf("Search: %s  (Enter close · Esc clear)", q)}
	}
	if strings.TrimSpace(m.Screen.SearchQuery) != "" {
		return []string{fmt.Sprintf("Search active: %q · %d match(es) · n/N · / reopen", m.Screen.SearchQuery, len(m.Screen.SearchHits))}
	}
	return nil
}
