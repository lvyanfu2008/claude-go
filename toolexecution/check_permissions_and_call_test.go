package toolexecution

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"goc/types"
)

type stubNamedTool struct {
	n string
}

func (s stubNamedTool) Name() string { return s.n }

func (stubNamedTool) Aliases() []string { return nil }

func (stubNamedTool) Call(
	ctx context.Context,
	toolUseID string,
	input json.RawMessage,
	tcx *ToolUseContext,
	canUseTool CanUseToolFn,
	assistant AssistantMeta,
	onProgress func(toolUseID string, data json.RawMessage),
) (*types.ToolRunResult, error) {
	return nil, nil
}

func TestCheckPermissionsAndCallTool_preToolHookDenies(t *testing.T) {
	ctx := context.Background()
	deps := ExecutionDeps{
		PreToolUseHook: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error {
			return errors.New("hook says no")
		},
	}
	ctx = WithExecutionDeps(ctx, deps)
	msgs, err := CheckPermissionsAndCallTool(ctx, stubNamedTool{n: "Bash"}, "tu-1", json.RawMessage(`{}`), &ToolUseContext{}, nil, AssistantMeta{UUID: "asst-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	inner := string(msgs[0].Message)
	if !strings.Contains(inner, "hook says no") {
		t.Fatalf("message: %s", inner)
	}
}

func TestStreamedCheckPermissionsAndCallTool_preToolHookYieldsUserRow(t *testing.T) {
	ctx := context.Background()
	deps := ExecutionDeps{
		RandomUUID: func() string { return "u-deny" },
		PreToolUseHook: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error {
			return errors.New("denied")
		},
	}
	ctx = WithExecutionDeps(ctx, deps)
	var got int
	for upd, err := range StreamedCheckPermissionsAndCallTool(ctx, stubNamedTool{n: "Read"}, "tu-2", json.RawMessage(`{}`), &ToolUseContext{}, nil, AssistantMeta{UUID: "asst-2"}) {
		if err != nil {
			t.Fatal(err)
		}
		if upd.Message != nil {
			got++
		}
	}
	if got != 1 {
		t.Fatalf("updates=%d", got)
	}
}

func TestCheckPermissionsAndCallTool_inputSchemaInvalid(t *testing.T) {
	raw := json.RawMessage(`[{"name":"echo_stub","input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]`)
	reg, err := NewJSONToolRegistry(raw)
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := reg.FindToolByName("echo_stub")
	if !ok {
		t.Fatal("missing tool")
	}
	ctx := WithExecutionDeps(context.Background(), ExecutionDeps{})
	msgs, err := CheckPermissionsAndCallTool(ctx, tool, "tu-x", json.RawMessage(`{}`), nil, nil, AssistantMeta{UUID: "asst-x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if !strings.Contains(string(msgs[0].Message), "echo_stub") {
		t.Fatalf("expected validation context: %s", msgs[0].Message)
	}
}

func TestCheckPermissionsAndCallTool_hookAllowRuleDeny(t *testing.T) {
	allow := AllowDecision()
	perm := types.EmptyToolPermissionContextData()
	b, err := json.Marshal(map[string][]string{"localSettings": {"Read"}})
	if err != nil {
		t.Fatal(err)
	}
	perm.AlwaysDenyRules = b
	types.NormalizeToolPermissionContextData(&perm)
	ctx := context.Background()
	deps := ExecutionDeps{
		PreToolHookPermission: &allow,
		ToolPermission:        &perm,
	}
	ctx = WithExecutionDeps(ctx, deps)
	msgs, err := CheckPermissionsAndCallTool(ctx, stubNamedTool{n: "Read"}, "tu-1", json.RawMessage(`{}`), &ToolUseContext{}, nil, AssistantMeta{UUID: "asst-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if !strings.Contains(string(msgs[0].Message), "denied") {
		t.Fatalf("body %s", msgs[0].Message)
	}
}

func TestCheckPermissionsAndCallTool_queryGateDeny(t *testing.T) {
	ctx := context.Background()
	deps := ExecutionDeps{
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
			return DenyDecision("policy"), nil
		},
	}
	ctx = WithExecutionDeps(ctx, deps)
	msgs, err := CheckPermissionsAndCallTool(ctx, stubNamedTool{n: "Read"}, "tu-1", json.RawMessage(`{}`), &ToolUseContext{}, nil, AssistantMeta{UUID: "asst-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if !strings.Contains(string(msgs[0].Message), "policy") {
		t.Fatalf("body %s", msgs[0].Message)
	}
}

func TestRunPreToolUseHooks_delegates(t *testing.T) {
	var saw string
	deps := ExecutionDeps{
		PreToolUseHook: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error {
			saw = toolName + ":" + toolUseID
			return nil
		},
	}
	if err := RunPreToolUseHooks(context.Background(), deps, "Skill", "id1", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if saw != "Skill:id1" {
		t.Fatalf("got %q", saw)
	}
}
