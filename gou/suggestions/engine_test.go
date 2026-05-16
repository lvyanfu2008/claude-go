package suggestions

import (
	"testing"
)

func TestExtractCompletionToken_SimpleAt(t *testing.T) {
	token, rng, matched := extractCompletionToken("hello @foo", 10)
	if !matched {
		t.Fatal("expected matched=true")
	}
	if token != "foo" {
		t.Errorf("expected token 'foo', got %q", token)
	}
	if rng.Start != 7 || rng.End != 10 {
		t.Errorf("expected range [7,10], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_PathLike(t *testing.T) {
	token, rng, matched := extractCompletionToken("@./src/comp", 11)
	if !matched {
		t.Fatal("expected matched=true")
	}
	if token != "./src/comp" {
		t.Errorf("expected token './src/comp', got %q", token)
	}
	if rng.Start != 1 || rng.End != 11 {
		t.Errorf("expected range [1,11], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_NoAtSymbol(t *testing.T) {
	token, _, matched := extractCompletionToken("hello world", 11)
	if matched {
		t.Fatal("expected matched=false for text without @")
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractCompletionToken_AtLineStart(t *testing.T) {
	token, rng, matched := extractCompletionToken("@foo bar", 4)
	if !matched {
		t.Fatal("expected matched=true")
	}
	if token != "foo" {
		t.Errorf("expected token 'foo', got %q", token)
	}
	if rng.Start != 1 || rng.End != 4 {
		t.Errorf("expected range [1,4], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_MidLineAt(t *testing.T) {
	token, rng, matched := extractCompletionToken("run @test.go now", 12)
	if !matched {
		t.Fatal("expected matched=true")
	}
	if token != "test.go" {
		t.Errorf("expected token 'test.go', got %q", token)
	}
	if rng.Start != 5 || rng.End != 12 {
		t.Errorf("expected range [5,12], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_BareAt(t *testing.T) {
	token, rng, matched := extractCompletionToken("@", 1)
	if !matched {
		t.Fatal("expected matched=true for bare @")
	}
	if token != "" {
		t.Errorf("expected empty token for bare @, got %q", token)
	}
	if rng.Start != 1 || rng.End != 1 {
		t.Errorf("expected range [1,1] for bare @, got [%d,%d]", rng.Start, rng.End)
	}
}

func TestEngine_DismissResetsOnTokenChange(t *testing.T) {
	fi := &FileIndex{entries: []string{"foo.go", "bar.go"}}
	engine := NewSuggestionEngine(fi)

	// First update
	result := engine.Update("@foo", 4)
	if result == nil || !result.HasResults {
		t.Fatal("expected results for @foo")
	}

	// Dismiss
	engine.Dismiss()

	// Same token should return nil
	result2 := engine.Update("@foo", 4)
	if result2 != nil {
		t.Fatal("expected nil after dismiss with same token")
	}

	// Different token should work again
	result3 := engine.Update("@bar", 4)
	if result3 == nil {
		t.Fatal("expected results after token change")
	}
}

func TestEngine_PathLikeDetection(t *testing.T) {
	fi := &FileIndex{entries: []string{"src/main.go"}}
	engine := NewSuggestionEngine(fi)

	// Path-like token triggers GetTopLevelPaths first (will fail on nonexistent dir), then falls back to index search
	result := engine.Update("@./nonexistent", 14)
	if result == nil {
		t.Fatal("expected non-nil result for path-like token (should fall back to index search)")
	}
}

func TestIsPathLikeToken(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"./foo", true},
		{"../bar", true},
		{"~/proj", true},
		{"/abs/path", true},
		{"src/file", false},
		{"agent-name", false},
	}
	for _, tt := range tests {
		got := isPathLikeToken(tt.token)
		if got != tt.expected {
			t.Errorf("isPathLikeToken(%q) = %v, want %v", tt.token, got, tt.expected)
		}
	}
}

func TestSearchAgents_FindsMatch(t *testing.T) {
	agents := []AgentDef{
		{Name: "Explore", DisplayName: "Explore Agent", Description: "Searches the codebase"},
		{Name: "Plan", DisplayName: "Plan Agent", Description: "Designs implementation plans"},
	}
	results := searchAgents(agents, "explore")
	if len(results) == 0 {
		t.Fatal("expected agent match for 'explore'")
	}
	if results[0].Value != "agent-Explore" {
		t.Errorf("expected 'agent-Explore', got %q", results[0].Value)
	}
	if results[0].Icon != "*" {
		t.Errorf("expected Icon '*', got %q", results[0].Icon)
	}
}
