package appstate

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSessionHooksState_empty(t *testing.T) {
	var s SessionHooksState
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{}` {
		t.Fatalf("%s", b)
	}
}

func TestSessionHooksState_sanitize(t *testing.T) {
	s := SessionHooksState{
		"sess1": {Hooks: map[string][]SessionHookMatcherSnapshot{
			"UserPromptSubmit": {{Matcher: "m", Hooks: nil}},
		}},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back SessionHooksState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	h := back["sess1"].Hooks["UserPromptSubmit"][0].Hooks
	if h == nil {
		t.Fatal("expected non-nil hooks slice")
	}
}

func TestSessionHookEntrySnapshot_roundTrip(t *testing.T) {
	e := SessionHookEntrySnapshot{Hook: json.RawMessage(`{"type":"command","command":"echo"}`)}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var back SessionHookEntrySnapshot
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if string(back.Hook) != string(e.Hook) {
		t.Fatalf("%s vs %s", back.Hook, e.Hook)
	}
}

func TestNormalizeAppState_skillImprovementUpdates(t *testing.T) {
	a := AppState{
		SkillImprovement: SkillImprovementState{
			Suggestion: &SkillImprovementSuggestion{SkillName: "s", Updates: nil},
		},
	}
	NormalizeAppState(&a)
	if a.SkillImprovement.Suggestion.Updates == nil {
		t.Fatal("want empty slice")
	}
	if !reflect.DeepEqual(a.SkillImprovement.Suggestion.Updates, []SkillImprovementUpdate{}) {
		t.Fatalf("%#v", a.SkillImprovement.Suggestion.Updates)
	}
}
