// Package layout provides terminal layout helpers aligned with useVirtualScroll height semantics:
// visual cell width ignores ANSI; wrap preserves escape sequences (charmbracelet/x/ansi).
package layout

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// MeasuredLine is one physical terminal row after wrap (plan: go-tui-message-stream-virtual-scroll §5.2).
type MeasuredLine struct {
	// VisualWidth is cell count of printable content (ANSI stripped).
	VisualWidth int
	// Content is the line as drawn (may include ANSI).
	Content string
}

// VisualWidth returns terminal cell width of s (ANSI codes not counted).
func VisualWidth(s string) int {
	if s == "" {
		return 0
	}
	return ansi.StringWidthWc(s)
}

// WrapForViewport wraps s to cols cells per line, preserving ANSI and wide chars.
// Mirrors the need for Ink/Yoga layout + render-node-to-output without splitting escapes.
func WrapForViewport(s string, cols int) string {
	if cols < 1 {
		return s
	}
	return ansi.HardwrapWc(s, cols, false)
}

// WrappedRowCount returns the number of terminal rows s occupies after WrapForViewport.
func WrappedRowCount(s string, cols int) int {
	if s == "" {
		return 0
	}
	w := WrapForViewport(s, cols)
	if w == "" {
		return 0
	}
	return strings.Count(w, "\n") + 1
}

// SplitMeasuredLines splits wrapped output into MeasuredLine records (optional profiling / hit-test).
func SplitMeasuredLines(wrapped string) []MeasuredLine {
	if wrapped == "" {
		return nil
	}
	lines := strings.Split(wrapped, "\n")
	out := make([]MeasuredLine, len(lines))
	for i, ln := range lines {
		out[i] = MeasuredLine{
			VisualWidth: VisualWidth(ln),
			Content:     ln,
		}
	}
	return out
}
