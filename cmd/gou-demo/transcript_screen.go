package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/conversation"
)

// gouDemoScreen mirrors TS Screen in src/screens/REPL.tsx ('prompt' | 'transcript').
type gouDemoScreen int

const (
	gouDemoScreenPrompt gouDemoScreen = iota
	gouDemoScreenTranscript
)

// frozenTranscriptSnapshot mirrors REPL.tsx useState frozenTranscriptState:
// { messagesLength, streamingToolUsesLength } (see handleEnterTranscript).
// gou-demo has no separate streamingToolUses slice in the store; length stays 0
// until a parity structure exists (Messages still freeze at toggle time).
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

func (m *model) scrollItemKeys() []string {
	n := m.transcriptEffectiveN()
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = conversation.ItemKey(m.store.Messages[i], m.store.ConversationID)
	}
	return keys
}

func (m *model) enterTranscriptScreen() {
	m.clearTranscriptSearchState()
	m.promptSavedScrollTop = m.scrollTop
	m.promptSavedSticky = m.sticky
	// TS: handleEnterTranscript sets frozen lengths; toggle handler also setShowAllInTranscript(false).
	m.transcriptFrozen = &frozenTranscriptSnapshot{
		MessagesLen:          len(m.store.Messages),
		StreamingToolUsesLen: 0,
	}
	m.transcriptShowAll = false
	m.transcriptDumpMode = false
	m.uiScreen = gouDemoScreenTranscript
	m.sticky = true
	m.scrollTop = 1 << 30
}

func (m *model) exitTranscriptScreen() {
	m.clearTranscriptSearchState()
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
}

// exitTranscriptScreenWithPostCmd restores the alternate screen after TS-style [ dump (Ink unwrap).
func (m *model) exitTranscriptScreenWithPostCmd() tea.Cmd {
	hadDump := m.transcriptDumpMode
	m.exitTranscriptScreen()
	if m.programUsesAltScreen && hadDump {
		return func() tea.Msg { return tea.EnterAltScreen() }
	}
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
	line := fmt.Sprintf("Transcript · %s toggle · ctrl+e %s · jk gG ctrl+udbf · / search · [ v · Esc/q/ctrl+c", toggle, showAllHint)
	if narrow {
		line = fmt.Sprintf("Transcript · %s · ctrl+e %s · jk · / · [ v · Esc", toggle, showAllHint)
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
