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
	raw := strings.TrimRight(line, "\r")
	if raw == "" {
		return line
	}
	// [IndentedWriteEditDiffLinesFromToolResultJSON] prefixes each row with two spaces for TUI
	// alignment; classify on the logical unified-diff line only, then re-attach the prefix so
	// true context lines (space-prefixed in unified format) stay correct.
	uiPrefix := ""
	logical := raw
	if strings.HasPrefix(logical, "  ") {
		uiPrefix = "  "
		logical = logical[2:]
	}
	if logical == "" {
		return line
	}
	render := func(st lipgloss.Style) string {
		return uiPrefix + st.Render(logical)
	}
	if strings.HasPrefix(logical, "--- ") || strings.HasPrefix(logical, "+++") || strings.HasPrefix(logical, "diff --git") {
		return render(lipgloss.NewStyle().Foreground(p.ToolMuted).Faint(true))
	}
	if strings.HasPrefix(logical, "@@") {
		return render(lipgloss.NewStyle().Foreground(theme.ToolWarning()).Faint(true))
	}
	if strings.HasPrefix(logical, "+") && !strings.HasPrefix(logical, "+++") {
		return render(lipgloss.NewStyle().Foreground(p.DiffAdd))
	}
	if strings.HasPrefix(logical, "-") && !strings.HasPrefix(logical, "---") {
		return render(lipgloss.NewStyle().Foreground(p.DiffDel))
	}
	if strings.Contains(logical, "unchanged lines") && strings.Contains(logical, "⋯") {
		return render(lipgloss.NewStyle().Foreground(p.ToolMuted).Faint(true))
	}
	return render(lipgloss.NewStyle().Foreground(p.ToolMuted))
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
