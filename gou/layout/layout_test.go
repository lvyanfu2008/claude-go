package layout

import (
	"strings"
	"testing"
)

func TestVisualWidth_ignoresANSI(t *testing.T) {
	const s = "\x1b[31mhi\x1b[0m"
	if VisualWidth(s) != 2 {
		t.Fatalf("got %d", VisualWidth(s))
	}
}

func TestWrappedRowCount_wideChars(t *testing.T) {
	// Two fullwidth chars often occupy 4 cells; with cols=2 that's 2 rows.
	s := "你好"
	n := WrappedRowCount(s, 2)
	if n < 2 {
		t.Fatalf("want >=2 rows, got %d", n)
	}
}

func TestWrapForViewport_preservesEscape(t *testing.T) {
	s := "\x1b[35m" + strings.Repeat("a", 20) + "\x1b[0m"
	w := WrapForViewport(s, 8)
	if !strings.Contains(w, "\x1b[35m") {
		t.Fatal("lost SGR")
	}
}
