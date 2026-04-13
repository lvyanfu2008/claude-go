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
	m.promptSavedScrollTop = m.scrollTop
	m.promptSavedSticky = m.sticky
	m.transcriptFreezeN = len(m.store.Messages)
	m.transcriptShowAll = false
	m.uiScreen = gouDemoScreenTranscript
	m.sticky = true
	m.scrollTop = 1 << 30
}

func (m *model) exitTranscriptScreen() {
	m.uiScreen = gouDemoScreenPrompt
	m.scrollTop = m.promptSavedScrollTop
	m.sticky = m.promptSavedSticky
	m.transcriptFreezeN = 0
}

// handleTranscriptKey returns true when the key was consumed (transcript mode only).
func (m *model) handleTranscriptKey(msg tea.KeyMsg) bool {
	if m.uiScreen != gouDemoScreenTranscript {
		return false
	}
	s := msg.String()
	switch s {
	case "ctrl+o":
		m.exitTranscriptScreen()
		return true
	case "ctrl+e":
		m.transcriptShowAll = !m.transcriptShowAll
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
	default:
		// Swallow typing so the prompt buffer is not mutated while browsing (TS has no prompt in transcript).
		return true
	}
}

func transcriptFooterLines(narrow, showAll bool) []string {
	toggle := "ctrl+o"
	showAllHint := "off"
	if showAll {
		showAllHint = "on"
	}
	line := fmt.Sprintf("Transcript · %s toggle · ctrl+e show-all %s · Esc/q/ctrl+c close", toggle, showAllHint)
	if narrow {
		line = fmt.Sprintf("Transcript · %s · ctrl+e %s · Esc", toggle, showAllHint)
	}
	return []string{line}
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
