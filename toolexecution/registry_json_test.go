package toolexecution

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

func TestNewJSONToolRegistry_CallEchoesInput(t *testing.T) {
	tools := json.RawMessage(`[{"name":"echo_stub","input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]`)
	reg, err := NewJSONToolRegistry(tools)
	if err != nil {
		t.Fatal(err)
	}
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "echo_stub", Input: json.RawMessage(`{"message":"hi"}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		ExecutionDeps{
			RandomUUID: func() string { return "rid" },
			Registry:   reg,
		},
		nil,
	)
	var body string
	for u := range ch {
		if u.Message != nil {
			body = string(u.Message.Message)
		}
	}
	if !strings.Contains(body, "message") || !strings.Contains(body, "hi") {
		t.Fatalf("expected echoed JSON in tool_result, got %s", body)
	}
}

func TestRunToolUseChan_schemaValidationError(t *testing.T) {
	tools := json.RawMessage(`[{"name":"echo_stub","input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]`)
	reg, err := NewJSONToolRegistry(tools)
	if err != nil {
		t.Fatal(err)
	}
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "echo_stub", Input: json.RawMessage(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		ExecutionDeps{RandomUUID: func() string { return "rid" }, Registry: reg},
		nil,
	)
	for u := range ch {
		if u.Message == nil {
			continue
		}
		if !strings.Contains(string(u.Message.Message), `"is_error":true`) {
			t.Fatalf("expected error tool_result: %s", u.Message.Message)
		}
	}
}

func TestRunToolUseChan_queryCanUseDeny(t *testing.T) {
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "Any", Input: json.RawMessage(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		ExecutionDeps{
			RandomUUID: func() string { return "rid" },
			QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
				return DenyDecision("no"), nil
			},
			InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
				return "should-not-run", false, nil
			},
		},
		nil,
	)
	for u := range ch {
		if u.Message != nil && strings.Contains(string(u.Message.Message), "should-not-run") {
			t.Fatal("invoke should not run after deny")
		}
	}
}
