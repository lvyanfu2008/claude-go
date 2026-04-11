package query

import (
	"context"
	"encoding/json"
	"testing"

	"goc/types"
)

func TestQueryReturnsCompleted(t *testing.T) {
	ctx := context.Background()
	p := QueryParams{
		Messages: []types.Message{
			{Type: types.MessageTypeUser, UUID: "u1", Message: json.RawMessage(`{"role":"user","content":"hi"}`)},
		},
		SystemPrompt: AsSystemPrompt([]string{"sys"}),
		UserContext:  map[string]string{},
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{},
		},
	}
	var last QueryYield
	var n int
	for y, err := range Query(ctx, p) {
		if err != nil {
			t.Fatal(err)
		}
		last = y
		n++
	}
	if n != 1 {
		t.Fatalf("yield count %d", n)
	}
	if last.Terminal == nil || last.Terminal.Reason != TerminalReasonCompleted {
		t.Fatalf("terminal %#v", last.Terminal)
	}
}

func TestQueryCallModelYieldsThenCompleted(t *testing.T) {
	ctx := context.Background()
	sentinel := types.Message{Type: types.MessageTypeAssistant, UUID: "a1"}
	deps := ProductionDeps()
	deps.CallModel = func(ctx context.Context, in *CallModelInput, emit func(QueryYield) bool) error {
		if ctx == nil {
			t.Fatal("nil ctx")
		}
		if len(in.Messages) < 1 {
			t.Fatalf("messages %d", len(in.Messages))
		}
		emit(QueryYield{Message: &sentinel})
		return nil
	}
	p := QueryParams{
		Messages: []types.Message{
			{Type: types.MessageTypeUser, UUID: "u1", Message: json.RawMessage(`{"role":"user","content":"hi"}`)},
		},
		SystemPrompt:   AsSystemPrompt([]string{"base"}),
		UserContext:    map[string]string{},
		SystemContext:  map[string]string{"k": "v"},
		ToolUseContext: types.ToolUseContext{Options: types.ToolUseContextOptionsData{}},
		Deps:           &deps,
	}
	var ys []QueryYield
	for y, err := range Query(ctx, p) {
		if err != nil {
			t.Fatal(err)
		}
		ys = append(ys, y)
	}
	if len(ys) != 2 {
		t.Fatalf("yields %d", len(ys))
	}
	if ys[0].Message == nil || ys[0].Message.UUID != "a1" {
		t.Fatalf("first %#v", ys[0])
	}
	if ys[1].Terminal == nil || ys[1].Terminal.Reason != TerminalReasonCompleted {
		t.Fatalf("second %#v", ys[1])
	}
}

func TestIsWithheldMaxOutputTokens(t *testing.T) {
	m := &types.Message{
		Type:    types.MessageTypeAssistant,
		Message: json.RawMessage(`{"role":"assistant","content":[],"apiError":"max_output_tokens"}`),
	}
	if !IsWithheldMaxOutputTokens(m) {
		t.Fatal("expected true")
	}
	if IsWithheldMaxOutputTokens(&types.Message{Type: types.MessageTypeUser}) {
		t.Fatal("expected false for user")
	}
}

func TestMissingToolResultUserMessages(t *testing.T) {
	am := types.Message{
		Type: types.MessageTypeAssistant,
		UUID: "asst-1",
		Message: json.RawMessage(`{"role":"assistant","content":[
			{"type":"tool_use","id":"toolu_1","name":"Bash","input":{}}
		]}`),
	}
	var got []types.Message
	for m := range MissingToolResultUserMessages([]types.Message{am}, "interrupted") {
		got = append(got, m)
	}
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Type != types.MessageTypeUser {
		t.Fatalf("type %s", got[0].Type)
	}
	if got[0].SourceToolAssistantUUID == nil || *got[0].SourceToolAssistantUUID != "asst-1" {
		t.Fatalf("source %#v", got[0].SourceToolAssistantUUID)
	}
}
