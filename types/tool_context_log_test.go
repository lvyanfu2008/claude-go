package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatProcessInputContextForLog_summary(t *testing.T) {
	tools := json.RawMessage(`[{"name":"Read"},{"name":"Skill"}]`)
	rc := &ProcessUserInputContextData{
		ToolUseContext: ToolUseContext{
			Options: ToolUseContextOptionsData{
				Commands: []Command{
					{CommandBase: CommandBase{Name: "a"}, Type: "prompt"},
					{CommandBase: CommandBase{Name: "b"}, Type: "prompt"},
				},
				MainLoopModel: "m1",
				Tools:         tools,
			},
			Messages: []Message{{Type: MessageTypeUser, UUID: "u1"}},
		},
	}
	b, err := FormatProcessInputContextForLog(rc, false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "commandsTotal") || !strings.Contains(s, `"a"`) {
		t.Fatalf("%s", s)
	}
	if !strings.Contains(s, "toolDefsTotal") || !strings.Contains(s, "Read") {
		t.Fatalf("%s", s)
	}
}

func TestFormatProcessInputContextForLog_nil(t *testing.T) {
	b, err := FormatProcessInputContextForLog(nil, false)
	if err != nil || string(b) != "null" {
		t.Fatalf("%s %v", b, err)
	}
}
