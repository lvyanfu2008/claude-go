package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Layout / resize reactions (extracted from [model.Update]).

func (m *model) handleUpdateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	oldCols := m.cols
	oldH := m.height
	m.width = msg.Width
	m.height = msg.Height
	m.cols = max(12, msg.Width-4)
	_ = m.pr.Update(msg)
	// Reserve two columns for the "> " prefix on the first line of the multiline input (see userInputViewWithPromptPrefix).
	m.pr.SetWidth(max(8, m.cols-2))
	if m.uiScreen == gouDemoScreenTranscript && oldCols > 0 && oldCols != m.cols {
		m.clearTranscriptSearchState()
	}
	// Always rebuild (not ScaleHeightCache only): message wrap width may be m.cols-1 when the TUI scrollbar strip is shown.
	if oldCols != m.cols || oldH != m.height || len(m.heightCache) == 0 {
		m.rebuildHeightCache()
	}
	if m.useMsgViewport && m.uiScreen == gouDemoScreenPrompt && !m.msgViewportFallback {
		m.vpNeedResizeContent = true
		m.lastVpContentSig = ""
		m.lastVpGeom = ""
	}
	return m, nil
}
