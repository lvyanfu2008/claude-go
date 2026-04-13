package main

import "testing"

func TestMessageScrollbarThumbBounds(t *testing.T) {
	start, length := messageScrollbarThumb(10, 100, 0)
	if start != 0 || length < 1 || length > 10 {
		t.Fatalf("top: start=%d len=%d", start, length)
	}
	start, length = messageScrollbarThumb(10, 100, 90)
	if start+length > 10 {
		t.Fatalf("tail overflow: start=%d len=%d", start, length)
	}
	start, length = messageScrollbarThumb(10, 10, 0)
	if start != 0 || length != 10 {
		t.Fatalf("no overflow: start=%d len=%d want 0,10", start, length)
	}
}

func TestJoinMessagePaneLinesWithScrollbarNoBar(t *testing.T) {
	s := joinMessagePaneLinesWithScrollbar([]string{"a", "b"}, 10, 2, 100, 0, 0)
	if s != "a\nb" {
		t.Fatalf("got %q", s)
	}
}
