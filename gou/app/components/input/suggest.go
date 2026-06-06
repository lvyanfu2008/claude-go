package input

import (
	"strings"

	"charm.land/lipgloss/v2"

	state "goc/gou/app/state"
	"goc/gou/suggestions"
)

// RenderSuggestions renders the @-mention autocomplete suggestion list above the prompt input.
// Returns empty string when no suggestions are visible.
func RenderSuggestions(deps Deps) string {
	if !deps.SuggVisible() || len(deps.Suggestions()) == 0 || deps.ScreenMode() != state.ScreenPrompt {
		return ""
	}
	width := deps.LayoutCols()
	if width < 40 {
		width = 40
	}
	maxVisible := 6
	sugg := deps.Suggestions()
	if len(sugg) < maxVisible {
		maxVisible = len(sugg)
	}
	// Center the visible window on selectedSuggIdx
	start := deps.SelectedSuggIdx() - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > len(sugg) {
		start = len(sugg) - maxVisible
	}
	if start < 0 {
		start = 0
	}

	var b strings.Builder
	// Title line
	title := lipgloss.NewStyle().Bold(true).Render("Suggestions  ") +
		lipgloss.NewStyle().Faint(true).Render("Tab/Enter accept  Esc dismiss")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')

	for i := start; i < len(sugg) && i < start+maxVisible; i++ {
		item := sugg[i]
		icon := item.Icon
		if icon == "" {
			switch item.Type {
			case suggestions.SuggestionTypeFile:
				icon = "F"
			case suggestions.SuggestionTypeDirectory:
				icon = "D"
			case suggestions.SuggestionTypeAgent:
				icon = "*"
			case suggestions.SuggestionTypeMcpResource:
				icon = "◇"
			}
		}
		line := "  " + icon + " " + item.Label
		if i == deps.SelectedSuggIdx() {
			line = lipgloss.NewStyle().Reverse(true).Render("  " + icon + " " + item.Label)
		}
		b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
		b.WriteByte('\n')
	}
	return b.String()
}
