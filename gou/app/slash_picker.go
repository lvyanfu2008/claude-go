package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"goc/types"
)

// slashPickerModel consolidates slash command picker state and rendering.
type slashPickerModel struct {
	commands   []types.Command
	loaded     bool
	userToggle bool
	selection  int
}

func newSlashPickerModel() *slashPickerModel {
	return &slashPickerModel{}
}

func (sp *slashPickerModel) SetCommands(cmds []types.Command) {
	sp.commands = cmds
	sp.loaded = true
}

// Visible reports whether the slash picker should be shown.
func (sp *slashPickerModel) Visible(value string, cursorRune int, isPrompt bool) bool {
	if !isPrompt || len(sp.commands) == 0 {
		return false
	}
	if sp.userToggle {
		return true
	}
	if shouldShowTSSlashList(value, cursorRune) {
		return true
	}
	return findMidInputSlashCommand(value, cursorRune) != nil
}

// ToggleUserManual toggles the F2 manual-list mode.
func (sp *slashPickerModel) ToggleUserManual() {
	if len(sp.commands) == 0 {
		return
	}
	sp.userToggle = !sp.userToggle
	sp.selection = 0
}

// Dismiss resets userToggle and selection.
func (sp *slashPickerModel) Dismiss() {
	sp.userToggle = false
	sp.selection = 0
}

// FilteredCommands returns ranked display names filtered by the input at cursor.
func (sp *slashPickerModel) FilteredCommands(value string, cursorRune int) []string {
	var q string
	if sp.userToggle {
		// F2 browse-all: empty query shows all non-hidden commands.
	} else if shouldShowTSSlashList(value, cursorRune) {
		q = slashFilterFromPrompt(value)
	} else if mid := findMidInputSlashCommand(value, cursorRune); mid != nil {
		q = mid.partial
	}
	return rankedSlashForQuery(sp.commands, q)
}

func (sp *slashPickerModel) NavUp() {
	if sp.selection > 0 {
		sp.selection--
	}
}

func (sp *slashPickerModel) NavDown(visible []string) {
	if sp.selection+1 < len(visible) {
		sp.selection++
	}
}

// ClampSelection ensures sp.selection is within [0, len(visible)).
func (sp *slashPickerModel) ClampSelection(visible []string) {
	if sp.selection >= len(visible) {
		if len(visible) == 0 {
			sp.selection = 0
		} else {
			sp.selection = len(visible) - 1
		}
	}
	if sp.selection < 0 {
		sp.selection = 0
	}
}

// View renders the slash picker block.
func (sp *slashPickerModel) View(visible []string, width, termHeight int, footerHint string) string {
	if len(visible) == 0 && !sp.userToggle {
		return ""
	}
	maxList := slashPickerMaxListRows(termHeight)
	var b strings.Builder
	rule := strings.Repeat("─", max(1, width))
	b.WriteString(lipgloss.NewStyle().Faint(true).Width(width).Render(rule))
	b.WriteByte('\n')
	title := lipgloss.NewStyle().Bold(true).Render("Slash commands  ") +
		lipgloss.NewStyle().Faint(true).Render(footerHint+"  F2  Esc  Tab  Enter run")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')
	start := 0
	idx := sp.selection
	if len(visible) > 0 && idx >= len(visible) {
		idx = len(visible) - 1
	}
	if len(visible) > 0 && idx >= maxList {
		start = idx - (maxList - 1)
		if start < 0 {
			start = 0
		}
	}
	indent := "  "
	for i := start; i < len(visible) && i < start+maxList; i++ {
		line := visible[i]
		if i == idx {
			b.WriteString(indent)
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(indent)
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if len(visible) == 0 {
		b.WriteString(indent)
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("(no matches)"))
	}
	return b.String()
}
