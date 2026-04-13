package query

import (
	"strings"
	"testing"
)

func TestStripSystemPromptDynamicBoundaryForAPI_nilEmpty(t *testing.T) {
	if StripSystemPromptDynamicBoundaryForAPI(nil) != nil {
		t.Fatal("nil in nil out")
	}
	if got := StripSystemPromptDynamicBoundaryForAPI(SystemPrompt{}); len(got) != 0 {
		t.Fatalf("empty: got %#v", []string(got))
	}
}

func TestStripSystemPromptDynamicBoundaryForAPI_noMarker(t *testing.T) {
	sp := AsSystemPrompt([]string{"A", "B"})
	got := StripSystemPromptDynamicBoundaryForAPI(sp)
	if strings.Join([]string(got), "\n\n") != "A\n\nB" {
		t.Fatalf("got %q", strings.Join([]string(got), "\n\n"))
	}
}

func TestStripSystemPromptDynamicBoundaryForAPI_embedded(t *testing.T) {
	b := systemPromptDynamicBoundary
	sp := AsSystemPrompt([]string{"foo", b, "bar"})
	got := StripSystemPromptDynamicBoundaryForAPI(sp)
	if strings.Join([]string(got), "\n\n") != "foo\n\nbar" {
		t.Fatalf("got %q", strings.Join([]string(got), "\n\n"))
	}
}

func TestStripSystemPromptDynamicBoundaryForAPI_oneString(t *testing.T) {
	b := systemPromptDynamicBoundary
	sp := AsSystemPrompt([]string{"foo\n\n" + b + "\n\nbar"})
	got := StripSystemPromptDynamicBoundaryForAPI(sp)
	if strings.Join([]string(got), "\n\n") != "foo\n\nbar" {
		t.Fatalf("got %q", strings.Join([]string(got), "\n\n"))
	}
}

func TestStripSystemPromptDynamicBoundaryForAPI_onlyBoundary(t *testing.T) {
	got := StripSystemPromptDynamicBoundaryForAPI(AsSystemPrompt([]string{systemPromptDynamicBoundary}))
	if got != nil {
		t.Fatalf("want nil, got %#v", []string(got))
	}
}
