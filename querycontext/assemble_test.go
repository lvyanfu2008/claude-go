package querycontext

import (
	"strings"
	"testing"
)

func TestAppendSystemContextParts_order(t *testing.T) {
	got := AppendSystemContextParts([]string{"base"}, map[string]string{
		"cacheBreaker": "[x]",
		"gitStatus":    "clean",
	})
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	if got[0] != "base" {
		t.Fatal(got[0])
	}
	// gitStatus before cacheBreaker (stable order)
	if got[1] != "gitStatus: clean\ncacheBreaker: [x]" {
		t.Fatalf("%q", got[1])
	}
}

func TestFormatUserContextReminder_tsShape(t *testing.T) {
	s := FormatUserContextReminder(map[string]string{
		"currentDate": "Today's date is 2026-01-01.",
	})
	if s == "" {
		t.Fatal("empty")
	}
	if s[:17] != "<system-reminder>" {
		t.Fatalf("prefix %q", s[:20])
	}
	if !strings.HasSuffix(s, "</system-reminder>\n") {
		t.Fatalf("suffix %q", s[len(s)-40:])
	}
	if !strings.Contains(s, "# currentDate\nToday's date is 2026-01-01.") {
		t.Fatalf("%q", s)
	}
	if !strings.Contains(s, "IMPORTANT: this context may or may not be relevant") {
		t.Fatal("missing IMPORTANT line")
	}
}
