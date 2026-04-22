// Package segdiff formats tool_result transcript segments with unified-diff–style colors (git-like).
// Lives outside cmd/gou-demo so it is not affected by global gitignore patterns matching "gou-demo".
package segdiff

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/layout"
	"goc/gou/message"
	"goc/gou/messagerow"
	"goc/gou/textutil"
	"goc/gou/theme"
)

// toolResultSegHeaderAndDiffBody splits messagerow tool_result segment text into the first-line
// header ("tool_result …") and a unified-diff body starting with "--- ".
func toolResultSegHeaderAndDiffBody(text string) (header, diffBody string, ok bool) {
	i := strings.Index(text, "\n")
	if i < 0 {
		return text, "", false
	}
	header = text[:i]
	diffBody = strings.TrimLeft(text[i+1:], "\n")
	if strings.HasPrefix(diffBody, "--- ") {
		return header, diffBody, true
	}
	return "", "", false
}

// FormatToolResultSegmentForTranscript renders SegToolResult: dim header + git-colored diff lines.
// Falls back to the legacy single-style block when the body is not a unified diff.
// baseMsgStyle matches gou-demo's [baseMsgStyle] (row background for user vs assistant).
func FormatToolResultSegmentForTranscript(
	seg messagerow.Segment,
	userRow, toolUseCtrlOHint bool,
	cols int,
	withHL func(string) string,
	baseMsgStyle func(userRow bool) lipgloss.Style,
) string {
	header, body, ok := toolResultSegHeaderAndDiffBody(seg.Text)
	if !ok || seg.ToolBodyOmitted {
		st := baseMsgStyle(userRow).Foreground(theme.DimMuted())
		if seg.IsToolError {
			st = baseMsgStyle(userRow).Foreground(theme.ToolError())
		}
		line := st.Render("↩ " + withHL(textutil.LinkifyOSC8(seg.Text)))
		line = wrapAndClampToolResultLines(line, cols, 10)
		if seg.ToolBodyOmitted && toolUseCtrlOHint {
			line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
		}
		return line
	}

	stMuted := baseMsgStyle(userRow).Foreground(theme.DimMuted())
	if seg.IsToolError {
		stMuted = baseMsgStyle(userRow).Foreground(theme.ToolError())
	}
	first := stMuted.Render("↩ " + withHL(textutil.LinkifyOSC8(header)))
	p := theme.ActivePalette()
	var b strings.Builder
	b.WriteString(first)
	for _, ln := range strings.Split(body, "\n") {
		b.WriteByte('\n')
		b.WriteString(message.FormatUnifiedDiffLineForDisplay(textutil.LinkifyOSC8(ln), seg.IsToolError, p))
	}
	out := wrapAndClampToolResultLines(b.String(), cols, 10)
	if seg.ToolBodyOmitted && toolUseCtrlOHint {
		out += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
	}
	return out
}

func wrapAndClampToolResultLines(s string, cols, maxLines int) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	if cols > 0 {
		s = layout.WrapForViewport(s, cols)
	}
	if maxLines < 1 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	lines = lines[:maxLines]
	lines[maxLines-1] += " ..."
	return strings.Join(lines, "\n")
}
