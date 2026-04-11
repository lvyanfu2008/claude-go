package query

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestBuildLocalTurnParams_messagesAndSystem(t *testing.T) {
	in := &CallModelInput{
		Messages: []types.Message{
			{Type: types.MessageTypeUser, UUID: "u1", Message: json.RawMessage(`{"role":"user","content":"hi"}`)},
		},
		SystemPrompt: AsSystemPrompt([]string{"block-a", "block-b"}),
		Tools:        json.RawMessage(`[]`),
		Cwd:          "/tmp/wd",
		ModelID:      "claude-sonnet-4-20250514",
	}
	p, err := buildLocalTurnParams(in, LocalTurnCallModelConfig{FetchSystemPromptIfEmpty: true}, "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if p.RequestID != "req-1" || p.Cwd != "/tmp/wd" || p.ModelID != "claude-sonnet-4-20250514" || !p.FetchSystemPromptIfEmpty {
		t.Fatalf("params %#v", p)
	}
	if p.System != "block-a\n\nblock-b" {
		t.Fatalf("system %q", p.System)
	}
	if !json.Valid(p.Messages) {
		t.Fatalf("messages invalid json %s", p.Messages)
	}
}
