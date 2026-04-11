package toolexecution

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

func TestRunToolUseChan_ruleDenyAfterQueryAllow(t *testing.T) {
	perm := types.EmptyToolPermissionContextData()
	b, _ := json.Marshal(map[string][]string{"localSettings": {"AnyTool"}})
	perm.AlwaysDenyRules = b
	types.NormalizeToolPermissionContextData(&perm)

	deps := ExecutionDeps{
		RandomUUID: func() string { return "r1" },
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			return AllowDecision(), nil
		},
		ToolPermission: &perm,
		InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
			t.Fatal("invoke after rule deny")
			return "", false, nil
		},
	}
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "AnyTool", Input: json.RawMessage(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		deps,
		nil,
	)
	for u := range ch {
		if u.Message != nil && u.Message.Type == types.MessageTypeUser {
			body := string(u.Message.Message)
			if !strings.Contains(body, "denied") && !strings.Contains(body, "Permission") {
				t.Fatalf("expected deny in body: %s", body)
			}
			return
		}
	}
	t.Fatal("expected message")
}
