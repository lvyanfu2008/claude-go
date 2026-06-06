package app

import (
	tea "charm.land/bubbletea/v2"
	state "goc/gou/app/state"
)

// Layout / resize reactions (extracted from [model.Update]).

func (m *model) handleUpdateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	oldCols := m.Layout.Cols
	oldH := m.Layout.Height
	m.Layout.Width = msg.Width
	m.Layout.Height = msg.Height
	m.Layout.Cols = max(12, msg.Width-4)
	_ = m.Input.PR.Update(msg)
	// Reserve two columns for the "> " prefix on the first line of the multiline input (see userInputViewWithPromptPrefix).
	m.Input.PR.SetWidth(max(8, m.Layout.Cols-2))
	if m.Screen.Mode == state.ScreenTranscript && oldCols > 0 && oldCols != m.Layout.Cols {
		m.clearTranscriptSearchState()
	}
	// Always rebuild (not ScaleHeightCache only): message wrap width may be m.Layout.Cols-1 when the TUI scrollbar strip is shown.
	if oldCols != m.Layout.Cols || oldH != m.Layout.Height || len(m.Scroll.HeightCache) == 0 {
		m.rebuildHeightCache()
	}
	if m.Viewport.Enabled && m.Screen.Mode == state.ScreenPrompt && !m.Viewport.Fallback {
		m.Viewport.NeedResizeContent = true
		m.Viewport.LastContentSig = ""
		m.Viewport.LastGeom = ""
	}
	return m, nil
}
