package permissionrules

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func testDenyContext(rules ...string) types.ToolPermissionContextData {
	b, err := json.Marshal(map[string][]string{"userSettings": rules})
	if err != nil {
		panic(err)
	}
	ctx := types.ToolPermissionContextData{
		Mode:                types.PermissionDefault,
		AlwaysDenyRules:     b,
		AlwaysAllowRules:    json.RawMessage(`{}`),
		AlwaysAskRules:      json.RawMessage(`{}`),
		AdditionalWorkingDirectories: json.RawMessage(`{}`),
	}
	types.NormalizeToolPermissionContextData(&ctx)
	return ctx
}

func TestGetDenyRuleForAgent(t *testing.T) {
	t.Parallel()
	ctx := testDenyContext()
	if GetDenyRuleForAgent(ctx, "Agent", "Explore") != nil {
		t.Fatal("expected no rule without deny list")
	}
	ctx = testDenyContext("Agent(Explore)")
	r := GetDenyRuleForAgent(ctx, "Agent", "Explore")
	if r == nil {
		t.Fatal("expected deny rule")
	}
	if r.Source != "userSettings" {
		t.Fatalf("source: %q", r.Source)
	}
	if GetDenyRuleForAgent(ctx, "Agent", "Research") != nil {
		t.Fatal("expected no rule for other type")
	}
	// Subagent name case
	if GetDenyRuleForAgent(ctx, "Agent", "explore") == nil {
		t.Fatal("expected case-insensitive match")
	}
	// TS legacy: Task(Explore) normalizes to Agent(Explore) in the rule
	ctxTask := testDenyContext("Task(Explore)")
	if GetDenyRuleForAgent(ctxTask, "Agent", "Explore") == nil {
		t.Fatal("expected deny from Task(Explore) rule")
	}
}

func TestFilterDeniedAgents(t *testing.T) {
	t.Parallel()
	type row struct{ agentType string }
	ctx := testDenyContext("Agent(Explore)")
	agents := []row{{"Explore"}, {`Research`}}
	out := FilterDeniedAgents(agents, func(r row) string { return r.agentType }, ctx, "Agent")
	if len(out) != 1 || out[0].agentType != "Research" {
		t.Fatalf("got %+v", out)
	}
}
