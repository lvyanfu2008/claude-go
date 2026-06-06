package update

import (
	tea "charm.land/bubbletea/v2"
	state "goc/gou/app/state"
)

func (d *Dispatcher) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	oldCols := d.deps.GetLayout().Cols
	oldH := d.deps.GetLayout().Height
	d.deps.GetLayout().Width = msg.Width
	d.deps.GetLayout().Height = msg.Height
	d.deps.GetLayout().Cols = max(12, msg.Width-4)
	_ = d.deps.GetInput().PR.Update(msg)
	// Reserve two columns for the "> " prefix on the first line of the multiline input (see userInputViewWithPromptPrefix).
	d.deps.GetInput().PR.SetWidth(max(8, d.deps.GetLayout().Cols-2))
	if d.deps.GetScreen().Mode == state.ScreenTranscript && oldCols > 0 && oldCols != d.deps.GetLayout().Cols {
		d.deps.ClearTranscriptSearchState()
	}
	// Always rebuild (not ScaleHeightCache only): message wrap width may be m.Layout.Cols-1 when the TUI scrollbar strip is shown.
	if oldCols != d.deps.GetLayout().Cols || oldH != d.deps.GetLayout().Height || len(d.deps.GetScroll().HeightCache) == 0 {
		d.deps.RebuildHeightCache()
	}
	if d.deps.GetViewport().Enabled && d.deps.GetScreen().Mode == state.ScreenPrompt && !d.deps.GetViewport().Fallback {
		d.deps.GetViewport().NeedResizeContent = true
		d.deps.GetViewport().LastContentSig = ""
		d.deps.GetViewport().LastGeom = ""
	}
	return d.deps.Model(), nil
}
