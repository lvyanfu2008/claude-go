package input

import (
	"strings"

	state "goc/gou/app/state"
)

// InputAreaHeight returns the height of the input area, including suggestions,
// status line, and the horizontal rule.
func InputAreaHeight(deps Deps) int {
	h := deps.InputLineCount()
	if deps.SuggVisible() && len(deps.Suggestions()) > 0 {
		visibleRows := min(6, len(deps.Suggestions()))
		h += 1 + visibleRows // title line + suggestion rows
	}
	if deps.ScreenMode() != state.ScreenTranscript {
		h++ // horizontal rule above input
	}
	if deps.ScreenMode() != state.ScreenTranscript && !deps.BuiltinStatusLineDisabled() {
		s := deps.BuiltinStatusLineView()
		if s != "" {
			h += strings.Count(s, "\n") + 1
		}
	}
	if h < 2 {
		h = 2
	}
	if h > 16 {
		h = 16
	}
	return h
}
