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
