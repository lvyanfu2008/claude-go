package app

import (
	"goc/gou/app/components/input"
	state "goc/gou/app/state"
	"goc/gou/suggestions"
)

// inputDeps adapts *model to input.Deps.
type inputDeps struct {
	m *model
}

func (d inputDeps) InputView() string                       { return d.m.Input.PR.View() }
func (d inputDeps) InputLineCount() int                      { return d.m.Input.PR.LineCount() }
func (d inputDeps) SuggVisible() bool                        { return d.m.Input.SuggVisible }
func (d inputDeps) Suggestions() []suggestions.ScoredItem    { return d.m.Input.Suggestions }
func (d inputDeps) SelectedSuggIdx() int                     { return d.m.Input.SelectedSuggIdx }
func (d inputDeps) ScreenMode() state.ScreenMode             { return d.m.Screen.Mode }
func (d inputDeps) BuiltinStatusLineView() string            { return d.m.builtinStatusLineView() }
func (d inputDeps) BuiltinStatusLineDisabled() bool          { return gouDemoBuiltinStatusLineDisabled() }
func (d inputDeps) LayoutCols() int                          { return d.m.Layout.Cols }
func (d inputDeps) LayoutHeight() int                        { return d.m.Layout.Height }
func (d inputDeps) SlashListVisible() bool                   { return d.m.slashListVisible() }
func (d inputDeps) SlashListSel() int                        { return d.m.slashListSel }
func (d inputDeps) VisibleSlashList() []string               { return d.m.visibleSlashList() }
func (d inputDeps) SlashListFooterHint() string              { return d.m.slashListFooterHint() }

// Compile-time check
var _ input.Deps = inputDeps{}
