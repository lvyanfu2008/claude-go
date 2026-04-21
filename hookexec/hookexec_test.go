package hookexec

import (
	"encoding/json"
	"testing"
)

func TestMatchesPattern_pipeExact(t *testing.T) {
	if !MatchesPattern("session_start", "session_start|compact") {
		t.Fatal("expected match in pipe list")
	}
	if MatchesPattern("nested_traversal", "session_start|compact") {
		t.Fatal("expected no match")
	}
	if !MatchesPattern("anything", "") {
		t.Fatal("empty matcher matches")
	}
	if !MatchesPattern("x", "*") {
		t.Fatal("star matches")
	}
}

func TestHasInstructionsLoaded(t *testing.T) {
	tab := HooksTable{
		"InstructionsLoaded": {{
			Matcher: "session_start",
			Hooks: []json.RawMessage{
				json.RawMessage(`{"type":"command","command":"true"}`),
			},
		}},
	}
	if !HasInstructionsLoaded(tab) {
		t.Fatal("expected true")
	}
	if HasInstructionsLoaded(HooksTable{}) {
		t.Fatal("expected false for empty")
	}
}

func TestParseHookJSONOutput_sessionStart(t *testing.T) {
	stdout := `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"hello"}}`
	got, err := ParseHookJSONOutput(stdout, "SessionStart")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestParseHookJSONOutput_omittedEventName(t *testing.T) {
	stdout := `{"hookSpecificOutput":{"additionalContext":"x"}}`
	got, _ := ParseHookJSONOutput(stdout, "SessionStart")
	if got != "x" {
		t.Fatalf("got %q", got)
	}
}
