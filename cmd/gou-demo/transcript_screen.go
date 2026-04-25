package main

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"goc/gou/conversation"
	"goc/gou/messagesview"
	"goc/types"
)

// gouDemoScreen mirrors TS Screen in src/screens/REPL.tsx ('prompt' | 'transcript').
type gouDemoScreen int

const (
	gouDemoScreenPrompt gouDemoScreen = iota
	gouDemoScreenTranscript
)

// frozenTranscriptSnapshot mirrors REPL.tsx useState frozenTranscriptState:
// { messagesLength, streamingToolUsesLength } (see handleEnterTranscript).
// streamingToolUsesLength is len(store.StreamingToolUses) at enter time (TS streamingToolUses.slice(0, n) in transcript).
type frozenTranscriptSnapshot struct {
	MessagesLen          int
	StreamingToolUsesLen int
}

func clampTranscriptFreeze(freezeN, nMsgs int) int {
	if nMsgs < 0 {
		nMsgs = 0
	}
	if freezeN < 0 {
		return 0
	}
	if freezeN > nMsgs {
		return nMsgs
	}
	return freezeN
}

func (m *model) transcriptEffectiveN() int {
	if m.uiScreen != gouDemoScreenTranscript {
		return len(m.store.Messages)
	}
	if m.transcriptFrozen == nil {
		return len(m.store.Messages)
	}
	return clampTranscriptFreeze(m.transcriptFrozen.MessagesLen, len(m.store.Messages))
}

// messagesForScroll returns UI-ordered messages (TS Messages.tsx pre-VirtualMessageList pipeline) for virtual scroll and transcript export.
func (m *model) messagesForScroll() []types.Message {
	var raw []types.Message
	if m.uiScreen == gouDemoScreenTranscript {
		n := m.transcriptEffectiveN()
		if n <= 0 {
			return nil
		}
		raw = slices.Clone(m.store.Messages[:n])
	} else {
		if len(m.store.Messages) == 0 {
			return nil
		}
		raw = slices.Clone(m.store.Messages)
	}
	return messagesview.MessagesForScrollList(raw, messagesview.ScrollListOpts{
		TranscriptMode:       m.uiScreen == gouDemoScreenTranscript,
		ShowAllInTranscript:  m.transcriptShowAll || m.transcriptDumpMode,
		VirtualScrollEnabled: !gouDemoVirtualScrollDisabled(),
		ResolvedToolUseIDs:   m.resolvedToolIDs,
	})
}

// messagePtrSliceForNewRenderer returns the same UI-ordered messages as [messagesForScroll] (Messages.tsx
// pipeline: progress dropped, null attachments dropped, reorder, grouping, etc.) as pointers for
// [MessageRendererIntegration] and [message.VirtualList]. This matches TS: progress is not a top-level
// message row; it is associated with tool use via progressMessagesForMessage / lookups.
func (m *model) messagePtrSliceForNewRenderer() []*types.Message {
	view := m.messagesForScroll()
	if len(view) == 0 {
		return nil
	}
	out := make([]*types.Message, len(view))
	for i := range view {
		out[i] = &view[i]
	}
	return out
}

// transcriptStreamToolScrollKey is a virtual-scroll key for in-transcript streaming tool rows (TS transcriptStreamingToolUses).
func transcriptStreamToolScrollKey(convID string, idx int) string {
	return fmt.Sprintf("gou-st-tool:%d:%s", idx, convID)
}

type GroupedStreamingTool struct {
	IsGroup     bool
	SearchCount int
	ReadCount   int
	ListCount   int
	Items       []conversation.StreamingToolUse
	Single      conversation.StreamingToolUse
}

func groupStreamingTools(uses []conversation.StreamingToolUse) []GroupedStreamingTool {
	var out []GroupedStreamingTool
	i := 0
	for i < len(uses) {
		tu := uses[i]
		name := strings.TrimSpace(tu.Name)
		switch name {
		case "Grep", "Glob", "Read", "View", "LS", "SemanticSearch":
			var group GroupedStreamingTool
			group.IsGroup = true
			j := i
			for j < len(uses) {
				n := strings.TrimSpace(uses[j].Name)
				if n == "Grep" || n == "Glob" || n == "SemanticSearch" {
					group.SearchCount++
				} else if n == "Read" || n == "View" {
					group.ReadCount++
				} else if n == "LS" {
					group.ListCount++
				} else {
					break
				}
				group.Items = append(group.Items, uses[j])
				j++
			}
			out = append(out, group)
			i = j
		default:
			out = append(out, GroupedStreamingTool{Single: tu})
			i++
		}
	}
	return out
}

// transcriptStreamingToolsForView returns grouped streaming tools while in transcript (REPL.tsx).
func (m *model) transcriptStreamingToolsForView() []GroupedStreamingTool {
	if m.uiScreen != gouDemoScreenTranscript || m.transcriptFrozen == nil {
		return nil
	}
	capN := m.transcriptFrozen.StreamingToolUsesLen
	if capN <= 0 {
		return nil
	}
	u := m.store.StreamingToolUses
	if len(u) > capN {
		u = u[:capN]
	}
	return groupStreamingTools(u)
}

func (m *model) scrollItemKeys() []string {
	msgView := m.messagesForScroll()
	keys := make([]string, 0, len(msgView)+len(m.transcriptStreamingToolsForView()))
	for i := range msgView {
		keys = append(keys, conversation.ItemKey(msgView[i], m.store.ConversationID))
	}
	keys = append(keys, m.transcriptStreamingToolScrollKeys()...)
	return keys
}

func (m *model) transcriptStreamingToolScrollKeys() []string {
	tools := m.transcriptStreamingToolsForView()
	out := make([]string, len(tools))
	for i := range tools {
		out[i] = transcriptStreamToolScrollKey(m.store.ConversationID, i)
	}
	return out
}

func (m *model) enterTranscriptScreen() tea.Cmd {
	m.clearSlashResultPanel()
	m.clearTranscriptSearchState()
	m.promptSavedScrollTop = m.scrollTop
	m.promptSavedSticky = m.sticky
	// TS: handleEnterTranscript sets frozen lengths; toggle handler also setShowAllInTranscript(false).
	m.transcriptFrozen = &frozenTranscriptSnapshot{
		MessagesLen:          len(m.store.Messages),
		StreamingToolUsesLen: len(m.store.StreamingToolUses),
	}
	m.transcriptShowAll = false
	m.transcriptDumpMode = false
	m.uiScreen = gouDemoScreenTranscript
	m.sticky = true
	m.scrollTop = 1 << 30
	m.pendingDelta = 0
	m.heightCache = nil
	m.rebuildHeightCache()
	return m.maybeTeaResetHistoryBrowseMouse()
}

func (m *model) exitTranscriptScreen() {
	m.clearTranscriptSearchState()
	m.suspendAltScreenForScrollbackDump = false
	m.uiScreen = gouDemoScreenPrompt
	m.scrollTop = m.promptSavedScrollTop
	m.sticky = m.promptSavedSticky
	// TS: handleExitTranscript / toggle clears frozenTranscriptState; exit also setShowAllInTranscript(false).
	m.transcriptFrozen = nil
	m.transcriptShowAll = false
	m.transcriptDumpMode = false
	m.transcriptEditorGen++
	m.transcriptEditorBusy = false
	m.transcriptEditorStatus = ""
	m.heightCache = nil
	m.pendingDelta = 0
	m.rebuildHeightCache()
	if m.useMsgViewport {
		m.lastVpContentSig = ""
		m.vpNeedResizeContent = true
	}
}

// exitTranscriptScreenWithPostCmd exits transcript mode; kept for call sites that expect a tea.Cmd return.
func (m *model) exitTranscriptScreenWithPostCmd() tea.Cmd {
	m.exitTranscriptScreen()
	return nil
}

func transcriptFooterLines(narrow, showAll, dumpMode bool) []string {
	toggle := "ctrl+o"
	showAllHint := "off"
	if showAll {
		showAllHint = "on"
	}
	if dumpMode {
		line := fmt.Sprintf("Transcript · %s toggle · [ dump · v $EDITOR · Esc/q/ctrl+c", toggle)
		if narrow {
			line = fmt.Sprintf("Transcript · %s · [ · v · Esc", toggle)
		}
		return []string{line}
	}
	line := fmt.Sprintf("Transcript · %s toggle · ctrl+l redraw · ctrl+e %s · jk gG ctrl+udbf · / search · [ v · Esc/q/ctrl+c", toggle, showAllHint)
	if narrow {
		line = fmt.Sprintf("Transcript · %s · ctrl+l · ctrl+e %s · jk · / · [ v · Esc", toggle, showAllHint)
	}
	return []string{line}
}

func transcriptChromeFootLines(m *model, narrow bool) []string {
	lines := transcriptFooterLines(narrow, m.transcriptShowAll, m.transcriptDumpMode)
	if extra := transcriptSearchStatusLines(m); len(extra) > 0 {
		lines = append(lines, extra...)
	}
	if s := strings.TrimSpace(m.transcriptEditorStatus); s != "" {
		lines = append(lines, s)
	}
	return lines
}

func joinFooterLines(lines []string, cols int) string {
	var b strings.Builder
	for i, ln := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if cols > 0 && len(ln) > cols {
			ln = ln[:max(0, cols-1)] + "…"
		}
		b.WriteString(ln)
	}
	return b.String()
}
