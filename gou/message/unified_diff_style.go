package message

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/theme"
)

// FormatUnifiedDiffLineForDisplay applies git-like ANSI colors to a single unified-diff line.
// Headers (--- / +++), hunk headers (@@), context (space), additions (+), deletions (−) are
// distinguished. When p is nil, returns line unchanged.
func FormatUnifiedDiffLineForDisplay(line string, toolResultIsError bool, p *theme.Palette) string {
	if p == nil {
		return line
	}
	if toolResultIsError {
		return lipgloss.NewStyle().Foreground(p.ToolError).Render(line)
	}
	t := strings.TrimRight(line, "\r")
	if t == "" {
		return line
	}
	if strings.HasPrefix(t, "--- ") || strings.HasPrefix(t, "+++") || strings.HasPrefix(t, "diff --git") {
		return lipgloss.NewStyle().Foreground(p.ToolMuted).Faint(true).Render(line)
	}
	if strings.HasPrefix(t, "@@") {
		return lipgloss.NewStyle().Foreground(theme.ToolWarning()).Faint(true).Render(line)
	}
	if strings.HasPrefix(t, "+") && !strings.HasPrefix(t, "+++") {
		return lipgloss.NewStyle().Foreground(p.DiffAdd).Render(line)
	}
	if strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "---") {
		return lipgloss.NewStyle().Foreground(p.DiffDel).Render(line)
	}
	if strings.Contains(t, "unchanged lines") && strings.Contains(t, "⋯") {
		return lipgloss.NewStyle().Foreground(p.ToolMuted).Faint(true).Render(line)
	}
	return lipgloss.NewStyle().Foreground(p.ToolMuted).Render(line)
}

// ApplyUnifiedDiffLineStyles maps each line through [FormatUnifiedDiffLineForDisplay].
func ApplyUnifiedDiffLineStyles(lines []string, toolResultIsError bool, p *theme.Palette) []string {
	if p == nil || len(lines) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i := range lines {
		out[i] = FormatUnifiedDiffLineForDisplay(lines[i], toolResultIsError, p)
	}
	return out
}
