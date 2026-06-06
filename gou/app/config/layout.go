package config

import (
	"strings"

	"goc/gou/layout"
)

// MessagePaneGutterCols is the uniform left indent for message pane body lines (alignment with wrap width).
const MessagePaneGutterCols = 2

// MessageWrapCols returns the available wrap width after accounting for the gutter.
func MessageWrapCols(cols int) int {
	if cols <= MessagePaneGutterCols+8 {
		return max(8, cols)
	}
	return cols - MessagePaneGutterCols
}

// ApplyMessagePaneGutter wraps block to (cols − gutter) and prefixes each line with two spaces.
func ApplyMessagePaneGutter(block string, cols int) string {
	if block == "" {
		return ""
	}
	wrapCols := MessageWrapCols(cols)
	wrapped := layout.WrapForViewport(block, wrapCols)
	prefix := strings.Repeat(" ", MessagePaneGutterCols)
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// MessagePaneGutterRowCount matches [ApplyMessagePaneGutter] line count for height cache parity.
func MessagePaneGutterRowCount(block string, cols int) int {
	g := ApplyMessagePaneGutter(block, cols)
	if g == "" {
		return 1
	}
	return max(1, strings.Count(g, "\n")+1)
}

// WrapHeadingForMessagePane wraps heading content to (MessageWrapCols − levelPad) so after [ApplyMessagePaneGutter]
// each physical line still includes the ATX level indent on continuations (not only the global two spaces).
func WrapHeadingForMessagePane(content string, levelPad string, cols int) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	innerW := MessageWrapCols(cols) - len(levelPad)
	if innerW < 8 {
		innerW = max(8, MessageWrapCols(cols)-2)
	}
	wrapped := layout.WrapForViewport(content, innerW)
	if levelPad == "" {
		return wrapped
	}
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = levelPad + lines[i]
	}
	return strings.Join(lines, "\n")
}
