package textutil

import (
	"strings"
	"testing"
)

func TestLinkifyOSC8_wrapsURL(t *testing.T) {
	s := "see https://example.com/path for info"
	out := LinkifyOSC8(s)
	if !strings.Contains(out, "\x1b]8;;https://example.com/path") {
		t.Fatalf("missing osc8: %q", out)
	}
	if !strings.Contains(out, "https://example.com/path") {
		t.Fatal("missing visible url")
	}
}
