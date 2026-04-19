package main

import "testing"

func TestMessageListMouseWheelStep(t *testing.T) {
	if messageListMouseWheelStep(0) != 1 {
		t.Fatal("vp 0 -> 1")
	}
	if got := messageListMouseWheelStep(24); got != 2 {
		t.Fatalf("24/12=2 got %d", got)
	}
	if got := messageListMouseWheelStep(11); got != 1 {
		t.Fatalf("11/12 floor got %d", got)
	}
}

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
