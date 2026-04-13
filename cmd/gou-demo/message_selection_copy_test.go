package main

import (
	"strings"
	"testing"
)

func TestSelectedPlainTextFromPaneLinesMultiRow(t *testing.T) {
	lines := []string{"hello world", "foo bar baz", "end"}
	// anchor (0,6) to (2,2) reading order
	got := selectedPlainTextFromPaneLines(lines, 20, 3, 0, 6, 2, 2)
	if got == "" {
		t.Fatal("empty")
	}
	if !strings.Contains(got, "foo bar baz") {
		t.Fatalf("got %q", got)
	}
}

func TestOsc52ClipboardNonEmpty(t *testing.T) {
	s := osc52ClipboardSequence("hi")
	if len(s) < 10 || !strings.Contains(s, "]52") {
		t.Fatalf("unexpected %q", s)
	}
}
