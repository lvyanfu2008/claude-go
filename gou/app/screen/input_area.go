package screen

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/theme"
)

// AboveInputRuleLine returns a faint full-width rule line between the context row and the multiline prompt.
func AboveInputRuleLine(cols int) string {
	if cols < 1 {
		cols = 40
	}
	rule := strings.Repeat("─", cols)
	return lipgloss.NewStyle().Faint(true).Foreground(theme.DimMuted()).Width(cols).Render(rule)
}
