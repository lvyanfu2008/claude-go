package main

import (
	"fmt"
	"strings"

	"goc/gou/conversation"
)

// gouDemoScreen mirrors TS Screen in src/screens/REPL.tsx ('prompt' | 'transcript').
type gouDemoScreen int

const (
	gouDemoScreenPrompt gouDemoScreen = iota
	gouDemoScreenTranscript
)

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
	return clampTranscriptFreeze(m.transcriptFreezeN, len(m.store.Messages))
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
	m.transcriptFreezeN = len(m.store.Messages)
	m.transcriptShowAll = false
	m.uiScreen = gouDemoScreenTranscript
	m.sticky = true
	m.scrollTop = 1 << 30
}

func (m *model) exitTranscriptScreen() {
	m.clearTranscriptSearchState()
	m.uiScreen = gouDemoScreenPrompt
	m.scrollTop = m.promptSavedScrollTop
	m.sticky = m.promptSavedSticky
	m.transcriptFreezeN = 0
}

func transcriptFooterLines(narrow, showAll bool) []string {
	toggle := "ctrl+o"
	showAllHint := "off"
	if showAll {
		showAllHint = "on"
	}
	line := fmt.Sprintf("Transcript · %s toggle · ctrl+e %s · jk gG ctrl+udbf · / search · Esc/q/ctrl+c", toggle, showAllHint)
	if narrow {
		line = fmt.Sprintf("Transcript · %s · ctrl+e %s · jk · / · Esc", toggle, showAllHint)
	}
	return []string{line}
}

func transcriptChromeFootLines(m *model, narrow bool) []string {
	lines := transcriptFooterLines(narrow, m.transcriptShowAll)
	if extra := transcriptSearchStatusLines(m); len(extra) > 0 {
		lines = append(lines, extra...)
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
