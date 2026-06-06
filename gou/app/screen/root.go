package screen

import (
	tea "charm.land/bubbletea/v2"
)

// rootPackage exposes the subset of Deps needed for wrapRootView.
type rootDeps interface {
	SuspendAltScreen() bool
	HistoryBrowseMouseOff() bool
}

// wrapRootView sets AltScreen and MouseMode on the tea.View based on env and model state.
func wrapRootView(v tea.View, d rootDeps) tea.View {
	v.AltScreen = AltScreenEnabled() && !d.SuspendAltScreen()
	if MouseCellMotionEnabled() {
		if d.HistoryBrowseMouseOff() {
			v.MouseMode = tea.MouseModeNone
		} else {
			v.MouseMode = tea.MouseModeCellMotion
		}
	}
	return v
}
