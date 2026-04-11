package toolexecution

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

func TestRunToolUseChan_askResolvedByAskResolver(t *testing.T) {
	tools := json.RawMessage(`[{"name":"echo_stub","input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]`)
	reg, err := NewJSONToolRegistry(tools)
	if err != nil {
		t.Fatal(err)
	}
	deps := ExecutionDeps{
		RandomUUID: func() string { return "r1" },
		Registry:   reg,
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			return AskDecision("approve me"), nil
		},
		AskResolver: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (PermissionDecision, error) {
			return AllowDecision(), nil
		},
	}
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "echo_stub", Input: json.RawMessage(`{"message":"z"}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		deps,
		nil,
	)
	var sawOK bool
	for u := range ch {
		if u.Message != nil && u.Message.Type == types.MessageTypeUser {
			body := string(u.Message.Message)
			if len(body) > 0 && !strings.Contains(body, "tool_use_error") {
				sawOK = true
			}
		}
	}
	if !sawOK {
		t.Fatal("expected successful tool_result path")
	}
}
