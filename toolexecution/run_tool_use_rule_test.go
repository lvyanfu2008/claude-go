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

type findToolRegistry struct {
	name string
	t    Tool
}

func (r findToolRegistry) FindToolByName(n string) (Tool, bool) {
	if n == r.name {
		return r.t, true
	}
	return nil, false
}

func TestRunToolUseChan_toolRuleCheckDenyAfterQueryAllow(t *testing.T) {
	tool := stubRuleCheckTool{
		stubNamedTool: stubNamedTool{n: "Custom"},
		fn: func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision {
			d := DenyDecision("subcommand denied")
			return &d
		},
	}
	deps := ExecutionDeps{
		RandomUUID: func() string { return "r1" },
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			return AllowDecision(), nil
		},
		Registry: findToolRegistry{name: "Custom", t: tool},
		InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
			t.Fatal("invoke after 1c deny")
			return "", false, nil
		},
	}
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "t1", Name: "Custom", Input: json.RawMessage(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst"},
		deps,
		nil,
	)
	for u := range ch {
		if u.Message != nil && u.Message.Type == types.MessageTypeUser {
			body := string(u.Message.Message)
			if !strings.Contains(body, "subcommand denied") {
				t.Fatalf("expected tool deny in body: %s", body)
			}
			return
		}
	}
	t.Fatal("expected message")
}
