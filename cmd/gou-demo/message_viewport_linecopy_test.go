package main

import (
	"strings"
	"testing"
)

func TestMsgLineCopyHighlightRange_ordersEndpoints(t *testing.T) {
	m := &model{msgLineCopyMode: true, msgLineCopyStart: 5, msgLineCopyEnd: 2}
	lo, hi, ok := m.msgLineCopyHighlightRange()
	if !ok || lo != 2 || hi != 5 {
		t.Fatalf("got lo=%d hi=%d ok=%v", lo, hi, ok)
	}
}

func TestApplyMsgLineCopyRowHighlight(t *testing.T) {
	lines := []string{"aa", "bb", "cc"}
	applyMsgLineCopyRowHighlight(lines, 10, 11, 12)
	if !strings.Contains(lines[1], "bb") {
		t.Fatalf("middle line should be highlighted: %q", lines[1])
	}
	if strings.Contains(lines[0], "▶") {
		t.Fatalf("first line should be untouched: %q", lines[0])
	}
}
