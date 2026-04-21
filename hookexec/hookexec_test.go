package hookexec

import (
	"encoding/json"
	"strings"
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

func TestDeriveMatchQuery_sessionStartAndTeammateIdle(t *testing.T) {
	mq, use := DeriveMatchQuery(map[string]any{"hook_event_name": "SessionStart", "source": "resume"})
	if !use || mq != "resume" {
		t.Fatalf("SessionStart: got %q use=%v", mq, use)
	}
	mq2, use2 := DeriveMatchQuery(map[string]any{"hook_event_name": "TeammateIdle"})
	if use2 || mq2 != "" {
		t.Fatalf("TeammateIdle: got %q use=%v", mq2, use2)
	}
}

func TestDeriveMatchQuery_fileChangedBasename(t *testing.T) {
	mq, use := DeriveMatchQuery(map[string]any{
		"hook_event_name": "FileChanged",
		"file_path":       "/a/b/c.md",
	})
	if !use || mq != "c.md" {
		t.Fatalf("got %q use=%v", mq, use)
	}
}

func TestAggregatePreCompactTS_joinsSuccessfulStdout(t *testing.T) {
	got := aggregatePreCompactTS([]OutsideReplCommandResult{
		{Command: "a", Succeeded: true, Output: "line1"},
		{Command: "b", Succeeded: true, Output: "line2"},
	})
	if got.Blocked {
		t.Fatal("blocked")
	}
	if !strings.Contains(got.NewCustomInstructions, "line1") || !strings.Contains(got.NewCustomInstructions, "line2") {
		t.Fatalf("instructions: %q", got.NewCustomInstructions)
	}
	if !strings.Contains(got.UserDisplayMessage, "PreCompact [a]") {
		t.Fatalf("display: %q", got.UserDisplayMessage)
	}
}

func TestAggregatePreCompactTS_blocked(t *testing.T) {
	got := aggregatePreCompactTS([]OutsideReplCommandResult{
		{Command: "x", Succeeded: false, Output: "", Blocked: true},
	})
	if !got.Blocked {
		t.Fatal("want Blocked")
	}
}

func TestAggregatePostCompactTS(t *testing.T) {
	got := aggregatePostCompactTS([]OutsideReplCommandResult{
		{Command: "c", Succeeded: true, Output: "ok"},
	})
	if !strings.Contains(got.UserDisplayMessage, "PostCompact [c]") {
		t.Fatalf("got %q", got.UserDisplayMessage)
	}
}

func TestCollectCommandHooks_emptyMatchQueryUsesAllMatchers(t *testing.T) {
	tab := HooksTable{
		"SessionStart": {
			{Matcher: "startup", Hooks: []json.RawMessage{json.RawMessage(`{"type":"command","command":"true"}`)}},
			{Matcher: "compact", Hooks: []json.RawMessage{json.RawMessage(`{"type":"command","command":"true"}`)}},
		},
	}
	h := CommandHooksForHookInput(tab, map[string]any{"hook_event_name": "SessionStart", "source": ""})
	if len(h) != 2 {
		t.Fatalf("want 2 hooks when source empty (TS no filter), got %d", len(h))
	}
}
