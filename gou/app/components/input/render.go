// Package input provides input area rendering (prompt prefix, @-mention suggestions,
// slash command list, and input layout) for the gou-demo TUI.
package input

import (
	state "goc/gou/app/state"
	"goc/gou/suggestions"
)

// Deps provides model-level dependencies for input rendering.
type Deps interface {
	InputView() string
	InputLineCount() int
	SuggVisible() bool
	Suggestions() []suggestions.ScoredItem
	SelectedSuggIdx() int
	ScreenMode() state.ScreenMode
	BuiltinStatusLineView() string
	BuiltinStatusLineDisabled() bool
	LayoutCols() int
	LayoutHeight() int
	SlashListVisible() bool
	SlashListSel() int
	VisibleSlashList() []string
	SlashListFooterHint() string
}

// InputRenderer renders the prompt input area (prefix, suggestions, slash picker).
type InputRenderer struct {
	deps Deps
}

// NewRenderer creates a new InputRenderer with the given dependencies.
func NewRenderer(deps Deps) *InputRenderer {
	return &InputRenderer{deps: deps}
}
