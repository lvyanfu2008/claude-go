package toolexecution

import (
	"context"
	"encoding/json"
	"testing"

	"goc/types"
)

func TestRuleBasedDecisionForTool_deny(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	b, err := json.Marshal(map[string][]string{"localSettings": {"Bash"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx.AlwaysDenyRules = b
	types.NormalizeToolPermissionContextData(&ctx)
	d := RuleBasedDecisionForTool("Bash", &ctx)
	if d == nil || d.Behavior != PermissionDeny {
		t.Fatalf("got %#v", d)
	}
}

func TestRuleBasedDecisionForTool_ask(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	b, err := json.Marshal(map[string][]string{"localSettings": {"Write"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx.AlwaysAskRules = b
	types.NormalizeToolPermissionContextData(&ctx)
	d := RuleBasedDecisionForTool("Write", &ctx)
	if d == nil || d.Behavior != PermissionAsk {
		t.Fatalf("got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_usesTcxToolPermission(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	b, _ := json.Marshal(map[string][]string{"localSettings": {"Read"}})
	ctx.AlwaysDenyRules = b
	types.NormalizeToolPermissionContextData(&ctx)
	tcx := &ToolUseContext{ToolPermission: &ctx}
	d := CheckRuleBasedPermissions(context.Background(), stubNamedTool{n: "Read"}, nil, tcx)
	if d == nil || d.Behavior != PermissionDeny {
		t.Fatalf("got %#v", d)
	}
}

type stubRuleCheckTool struct {
	stubNamedTool
	fn func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision
}

func (s stubRuleCheckTool) CheckPermissionsFromRules(ctx context.Context, input json.RawMessage, tcx *ToolUseContext) *PermissionDecision {
	if s.fn != nil {
		return s.fn(ctx, input, tcx)
	}
	return nil
}

func TestRuleBasedDecisionForTool_nilPermNoRules(t *testing.T) {
	if d := RuleBasedDecisionForTool("Bash", nil); d != nil {
		t.Fatalf("expected nil, got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_toolDeny1c(t *testing.T) {
	tool := stubRuleCheckTool{
		stubNamedTool: stubNamedTool{n: "Bash"},
		fn: func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision {
			d := DenyDecision("blocked by tool check")
			return &d
		},
	}
	d := CheckRuleBasedPermissions(context.Background(), tool, json.RawMessage(`{}`), &ToolUseContext{})
	if d == nil || d.Behavior != PermissionDeny || d.Message != "blocked by tool check" {
		t.Fatalf("got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_toolAskWithoutKindDropped(t *testing.T) {
	tool := stubRuleCheckTool{
		stubNamedTool: stubNamedTool{n: "Bash"},
		fn: func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision {
			a := AskDecision("generic ask")
			return &a
		},
	}
	if d := CheckRuleBasedPermissions(context.Background(), tool, nil, &ToolUseContext{}); d != nil {
		t.Fatalf("expected nil (TS null), got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_toolAskRuleContent1f(t *testing.T) {
	tool := stubRuleCheckTool{
		stubNamedTool: stubNamedTool{n: "Bash"},
		fn: func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision {
			a := AskRuleContentDecision("npm publish")
			return &a
		},
	}
	d := CheckRuleBasedPermissions(context.Background(), tool, nil, &ToolUseContext{})
	if d == nil || d.Behavior != PermissionAsk || d.AskKind != PermissionAskKindRuleContent {
		t.Fatalf("got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_bashWholeToolAskBypass1b(t *testing.T) {
	perm := types.EmptyToolPermissionContextData()
	b, err := json.Marshal(map[string][]string{"localSettings": {"Bash"}})
	if err != nil {
		t.Fatal(err)
	}
	perm.AlwaysAskRules = b
	types.NormalizeToolPermissionContextData(&perm)
	b1b := &BashSandboxRule1b{SandboxingEnabled: true, AutoAllowWholeToolAskWhenSandboxed: true}
	tcx := &ToolUseContext{ToolPermission: &perm, BashSandboxRule1b: b1b}
	tool := stubNamedTool{n: BashToolName}
	d := CheckRuleBasedPermissions(context.Background(), tool, json.RawMessage(`{"command":"ls -la"}`), tcx)
	if d != nil {
		t.Fatalf("expected nil (1b bypass), got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_bashWholeToolAskNoBypassWithoutSandboxFlags(t *testing.T) {
	perm := types.EmptyToolPermissionContextData()
	b, _ := json.Marshal(map[string][]string{"localSettings": {"Bash"}})
	perm.AlwaysAskRules = b
	types.NormalizeToolPermissionContextData(&perm)
	tcx := &ToolUseContext{ToolPermission: &perm}
	tool := stubNamedTool{n: BashToolName}
	d := CheckRuleBasedPermissions(context.Background(), tool, json.RawMessage(`{"command":"ls"}`), tcx)
	if d == nil || d.Behavior != PermissionAsk {
		t.Fatalf("got %#v", d)
	}
}

func TestCheckRuleBasedPermissions_toolAskSafety1g(t *testing.T) {
	tool := stubRuleCheckTool{
		stubNamedTool: stubNamedTool{n: "Write"},
		fn: func(context.Context, json.RawMessage, *ToolUseContext) *PermissionDecision {
			a := AskSafetyCheckDecision(".git is protected")
			return &a
		},
	}
	d := CheckRuleBasedPermissions(context.Background(), tool, nil, &ToolUseContext{})
	if d == nil || d.Behavior != PermissionAsk || d.AskKind != PermissionAskKindSafetyCheck {
		t.Fatalf("got %#v", d)
	}
}
