package toolexecution

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResolveHookPermissionDecision_queryGateDeny(t *testing.T) {
	dec, _, err := ResolveHookPermissionDecision(context.Background(), ResolveHookPermissionInput{
		Tool:      stubNamedTool{n: "Read"},
		Input:     json.RawMessage(`{}`),
		TCX:       &ToolUseContext{},
		ToolUseID: "id1",
		QueryGate: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			return DenyDecision("blocked"), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Behavior != PermissionDeny || dec.Message != "blocked" {
		t.Fatalf("got %+v", dec)
	}
}

func TestResolveHookPermissionDecision_hookAllowNoRuleUsesHook(t *testing.T) {
	allow := AllowDecision()
	dec, _, err := ResolveHookPermissionDecision(context.Background(), ResolveHookPermissionInput{
		HookPermission: &allow,
		Tool:             stubNamedTool{n: "Read"},
		Input:            json.RawMessage(`{}`),
		TCX:              &ToolUseContext{},
		ToolUseID:        "id1",
		QueryGate: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			t.Fatal("gate should not run when rule check is nil and no require flags")
			return AllowDecision(), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Behavior != PermissionAllow {
		t.Fatalf("got %+v", dec)
	}
}
