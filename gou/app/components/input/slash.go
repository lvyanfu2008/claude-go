package input

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// slashPickerMaxListRows returns the max number of list rows for the slash picker.
func SlashPickerMaxListRows(termHeight int) int {
	if termHeight < 1 {
		termHeight = 24
	}
	// keep modest so message pane stays primary
	return min(12, max(3, termHeight/4))
}

// slashPickerListRows returns the number of list body lines shown (0 if none, 1 for empty hint).
func SlashPickerListRows(vis []string, maxListRows int) int {
	if maxListRows < 1 {
		maxListRows = 1
	}
	if len(vis) == 0 {
		return 1
	}
	if len(vis) < maxListRows {
		return len(vis)
	}
	return maxListRows
}

// RenderSlashPicker draws a full-width block directly below the input: separator rule, then
// title + left-aligned list (not a corner overlay).
func RenderSlashPicker(deps Deps, width, termHeight int) string {
	if !deps.SlashListVisible() {
		return ""
	}
	if width < 1 {
		width = 40
	}
	vis := deps.VisibleSlashList()
	maxList := SlashPickerMaxListRows(termHeight)
	var b strings.Builder
	rule := strings.Repeat("─", max(1, width))
	b.WriteString(lipgloss.NewStyle().Faint(true).Width(width).Render(rule))
	b.WriteByte('\n')
	title := lipgloss.NewStyle().Bold(true).Render("Slash commands  ") +
		lipgloss.NewStyle().Faint(true).Render(deps.SlashListFooterHint() + "  F2  Esc  Tab  Enter run")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')
	start := 0
	idx := deps.SlashListSel()
	if len(vis) > 0 && idx >= len(vis) {
		idx = len(vis) - 1
	}
	if len(vis) > 0 && idx >= maxList {
		start = idx - (maxList - 1)
		if start < 0 {
			start = 0
		}
	}
	indent := "  "
	for i := start; i < len(vis) && i < start+maxList; i++ {
		line := vis[i]
		if i == idx {
			b.WriteString(indent)
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(indent)
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if len(vis) == 0 {
		b.WriteString(indent)
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("(no matches)"))
	}
	return b.String()
}
