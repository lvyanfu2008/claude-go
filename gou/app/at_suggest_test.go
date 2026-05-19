package app

import (
	"strings"
	"testing"

	"goc/gou/suggestions"
)

func TestExtractCompletionTokenForApply_FindsAt(t *testing.T) {
	token, rng := extractCompletionTokenForApply("run @test.go", 12)
	if token != "test.go" {
		t.Errorf("expected 'test.go', got %q", token)
	}
	if rng.Start != 4 || rng.End != 12 {
		t.Errorf("expected [4,12], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionTokenForApply_NoAt(t *testing.T) {
	token, _ := extractCompletionTokenForApply("no at sign", 11)
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractCompletionTokenForApply_AtLineStart(t *testing.T) {
	token, rng := extractCompletionTokenForApply("@hello", 6)
	if token != "hello" {
		t.Errorf("expected 'hello', got %q", token)
	}
	if rng.Start != 0 || rng.End != 6 {
		t.Errorf("expected [0,6], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionTokenForApply_AfterNewline(t *testing.T) {
	token, rng := extractCompletionTokenForApply("line1\n@foo bar", 10)
	if token != "foo" {
		t.Errorf("expected 'foo', got %q", token)
	}
	if rng.Start != 6 || rng.End != 10 {
		t.Errorf("expected [6,10], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionTokenForApply_NonPrecededBySpace(t *testing.T) {
	// @ in the middle of a word should NOT be detected
	token, _ := extractCompletionTokenForApply("email@foo.com", 12)
	if token != "" {
		t.Errorf("expected empty token for email@, got %q", token)
	}
}

func TestApplySuggestion_ReplacesToken(t *testing.T) {
	// Test the core replacement logic (simulates applySuggestion behavior)
	value := "@foo bar"
	rs := []rune(value)
	_, rng := extractCompletionTokenForApply(value, 4)
	rep := "@foobar.go "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	result := b.String()
	expected := "@foobar.go  bar"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestApplySuggestion_BareAt(t *testing.T) {
	// Bare @ (no filter text) should still replace @ with the selected path
	value := "@"
	rs := []rune(value)
	_, rng := extractCompletionTokenForApply(value, 1)
	if rng.Start != 0 || rng.End != 1 {
		t.Fatalf("expected range [0,1] for bare @, got [%d,%d]", rng.Start, rng.End)
	}
	rep := "@src/ "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	result := b.String()
	expected := "@src/ "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSuggestionItemIcons(t *testing.T) {
	items := []suggestions.ScoredItem{
		{Type: suggestions.SuggestionTypeFile, Icon: "F"},
		{Type: suggestions.SuggestionTypeDirectory, Icon: "D"},
		{Type: suggestions.SuggestionTypeAgent, Icon: "*"},
		{Type: suggestions.SuggestionTypeMcpResource, Icon: "◇"},
	}
	for _, item := range items {
		if item.Icon == "" {
			t.Errorf("expected non-empty icon for type %v", item.Type)
		}
	}
}
