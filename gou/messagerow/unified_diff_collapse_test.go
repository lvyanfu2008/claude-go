package messagerow

import (
	"strings"
	"testing"
)

func TestCollapseUnifiedDiffContextLines_trimsMiddle(t *testing.T) {
	lines := []string{
		"-old",
		" 1", " 2", " 3", " 4", " 5", " 6", " 7", " 8", " 9", " 10",
		"+new",
	}
	out := CollapseUnifiedDiffContextLines(lines, 2)
	s := strings.Join(out, "\n")
	if !strings.Contains(s, "unchanged lines") {
		t.Fatalf("expected omission marker, got:\n%s", s)
	}
	if len(out) >= len(lines) {
		t.Fatalf("expected shorter output, got len %d vs %d", len(out), len(lines))
	}
}
