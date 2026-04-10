package tscontext

import (
	"encoding/json"
	"testing"
)

func TestSnapshot_JSON_roundtrip(t *testing.T) {
	raw := `{
		"defaultSystemPrompt": ["a", "b"],
		"userContext": {"currentDate": "x"},
		"systemContext": {"gitStatus": "y"},
		"commands": [{"type":"prompt","name":"n","description":"d"}],
		"tools": [{"name":"t","description":"d","input_schema":{"type":"object"}}],
		"mainLoopModel": "m",
		"skillToolCommands": [{"type":"prompt","name":"skill1","description":"s"}],
		"slashCommandToolSkills": [],
		"agents": []
	}`
	var s Snapshot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.DefaultSystemPrompt) != 2 || s.DefaultSystemPrompt[0] != "a" {
		t.Fatalf("DefaultSystemPrompt %#v", s.DefaultSystemPrompt)
	}
	if s.UserContext["currentDate"] != "x" {
		t.Fatalf("UserContext %#v", s.UserContext)
	}
	if s.MainLoopModel != "m" {
		t.Fatalf("MainLoopModel %q", s.MainLoopModel)
	}
	if len(s.Commands) < 10 || len(s.Tools) < 10 {
		t.Fatalf("raw slices: commands=%d tools=%d", len(s.Commands), len(s.Tools))
	}
	if len(s.SkillToolCommands) < 20 {
		t.Fatalf("skillToolCommands: %s", s.SkillToolCommands)
	}
}
