package main

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/messagerow"
	"goc/types"
)

func (m *model) clearTranscriptSearchState() {
	m.transcriptSearchOpen = false
	m.transcriptSearchQuery = ""
	m.transcriptSearchHits = nil
	m.transcriptSearchCursor = 0
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

func (m *model) rebuildTranscriptSearchMatches() {
	n := m.transcriptEffectiveN()
	q := strings.TrimSpace(m.transcriptSearchQuery)
	if q == "" {
		m.transcriptSearchHits = nil
		m.transcriptSearchCursor = 0
		return
	}
	needle := strings.ToLower(q)
	var hits []int
	for i := 0; i < n; i++ {
		if strings.Contains(plainMessageSearchText(m.store.Messages[i]), needle) {
			hits = append(hits, i)
		}
	}
	m.transcriptSearchHits = hits
	if len(hits) == 0 {
		m.transcriptSearchCursor = 0
		return
	}
	if m.transcriptSearchCursor >= len(hits) {
		m.transcriptSearchCursor = 0
	}
	m.scrollTranscriptToMessageIndex(hits[m.transcriptSearchCursor])
}

func (m *model) scrollTranscriptToMessageIndex(msgIdx int) {
	keys := m.scrollItemKeys()
	if msgIdx < 0 || msgIdx >= len(keys) {
		return
	}
	off := 0
	for i := 0; i < msgIdx; i++ {
		off += m.heightCache[keys[i]]
	}
	m.scrollTop = off
	m.sticky = false
}

func (m *model) transcriptSearchStep(delta int) {
	h := m.transcriptSearchHits
	if len(h) == 0 {
		return
	}
	m.transcriptSearchCursor = (m.transcriptSearchCursor + delta + len(h)) % len(h)
	m.scrollTranscriptToMessageIndex(h[m.transcriptSearchCursor])
}

func (m *model) handleTranscriptSearchBarKey(msg tea.KeyMsg) bool {
	if !m.transcriptSearchOpen {
		return false
	}
	s := msg.String()
	switch s {
	case "esc":
		m.clearTranscriptSearchState()
		return true
	case "enter":
		m.transcriptSearchOpen = false
		return true
	case "backspace", "ctrl+h":
		if m.transcriptSearchQuery != "" {
			r := []rune(m.transcriptSearchQuery)
			if len(r) > 0 {
				m.transcriptSearchQuery = string(r[:len(r)-1])
				m.rebuildTranscriptSearchMatches()
			}
		}
		return true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		m.transcriptSearchQuery += string(msg.Runes)
		m.rebuildTranscriptSearchMatches()
		return true
	}
	return false
}

// handleTranscriptKey returns true when the key was consumed (transcript mode only).
func (m *model) handleTranscriptKey(msg tea.KeyMsg) bool {
	if m.uiScreen != gouDemoScreenTranscript {
		return false
	}
	if m.handleTranscriptSearchBarKey(msg) {
		return true
	}
	if !m.transcriptSearchOpen {
		if msg.String() == "/" {
			m.transcriptSearchOpen = true
			m.transcriptSearchQuery = ""
			m.rebuildTranscriptSearchMatches()
			return true
		}
		if strings.TrimSpace(m.transcriptSearchQuery) != "" {
			switch msg.String() {
			case "n":
				m.transcriptSearchStep(1)
				return true
			case "N":
				m.transcriptSearchStep(-1)
				return true
			}
		}
	}
	s := msg.String()
	switch s {
	case "ctrl+o":
		m.exitTranscriptScreen()
		return true
	case "ctrl+e":
		m.transcriptShowAll = !m.transcriptShowAll
		m.rebuildHeightCache()
		return true
	case "esc", "q", "ctrl+c":
		m.exitTranscriptScreen()
		return true
	case "up":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-1)
		return true
	case "down":
		m.sticky = false
		m.scrollTop += 1
		return true
	case "pgup":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
		return true
	case "pgdown":
		m.sticky = false
		m.scrollTop += listViewportH(m) / 2
		return true
	case "end":
		m.sticky = true
		m.scrollTop = 1 << 30
		return true
	// TS ScrollKeybindingHandler modalPagerAction (transcript, no prompt): j/k/g/G, less ctrl+u/d/b/f.
	// Not active while search bar is open (bare letters go into the query).
	case "j":
		m.sticky = false
		m.scrollTop += 1
		return true
	case "k":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-1)
		return true
	case "g":
		m.sticky = false
		m.scrollTop = 0
		return true
	case "G", "shift+g":
		m.sticky = true
		m.scrollTop = 1 << 30
		return true
	case "ctrl+u":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
		return true
	case "ctrl+d":
		m.sticky = false
		m.scrollTop += listViewportH(m) / 2
		return true
	case "ctrl+b":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m))
		return true
	case "ctrl+f":
		m.sticky = false
		m.scrollTop += listViewportH(m)
		return true
	case "b":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m))
		return true
	default:
		return true
	}
}

func transcriptSearchStatusLines(m *model) []string {
	if m.uiScreen != gouDemoScreenTranscript {
		return nil
	}
	if m.transcriptSearchOpen {
		q := m.transcriptSearchQuery
		if len(q) > 60 {
			q = q[:57] + "…"
		}
		return []string{fmt.Sprintf("Search: %s  (Enter close · Esc clear)", q)}
	}
	if strings.TrimSpace(m.transcriptSearchQuery) != "" {
		return []string{fmt.Sprintf("Search active: %q · %d match(es) · n/N · / reopen", m.transcriptSearchQuery, len(m.transcriptSearchHits))}
	}
	return nil
}
