package permissionrules

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func makeContext(denyRules, askRules []string) types.ToolPermissionContextData {
	ctx := types.EmptyToolPermissionContextData()
	if len(denyRules) > 0 {
		b, _ := json.Marshal(map[string][]string{"localSettings": denyRules})
		ctx.AlwaysDenyRules = b
	}
	if len(askRules) > 0 {
		b, _ := json.Marshal(map[string][]string{"localSettings": askRules})
		ctx.AlwaysAskRules = b
	}
	return ctx
}

func TestGetDenyRuleForTool(t *testing.T) {
	t.Parallel()
	tool := func(name string) types.ToolSpec { return types.ToolSpec{Name: name} }

	t.Run("no rules", func(t *testing.T) {
		ctx := makeContext(nil, nil)
		if GetDenyRuleForTool(ctx, tool("Bash")) != nil {
			t.Fatal("expected nil")
		}
	})
	t.Run("matching deny", func(t *testing.T) {
		ctx := makeContext([]string{"Bash"}, nil)
		r := GetDenyRuleForTool(ctx, tool("Bash"))
		if r == nil || r.RuleValue.ToolName != "Bash" {
			t.Fatalf("deny Bash: %#v", r)
		}
	})
	t.Run("non-matching", func(t *testing.T) {
		ctx := makeContext([]string{"Bash"}, nil)
		if GetDenyRuleForTool(ctx, tool("Read")) != nil {
			t.Fatal("expected nil for Read")
		}
	})
	t.Run("rule with content not whole-tool deny", func(t *testing.T) {
		ctx := makeContext([]string{"Bash(rm -rf)"}, nil)
		if GetDenyRuleForTool(ctx, tool("Bash")) != nil {
			t.Fatal("content rule must not match whole Bash")
		}
	})
}

func TestGetAskRuleForTool(t *testing.T) {
	t.Parallel()
	tool := types.ToolSpec{Name: "Write"}
	ctx := makeContext(nil, []string{"Write"})
	if GetAskRuleForTool(ctx, tool) == nil {
		t.Fatal("expected ask rule")
	}
	ctx2 := makeContext(nil, []string{"Write"})
	if GetAskRuleForTool(ctx2, types.ToolSpec{Name: "Bash"}) != nil {
		t.Fatal("expected nil")
	}
}

func TestFilterToolsByDenyRules(t *testing.T) {
	t.Parallel()
	ctx := makeContext([]string{"Bash", "Write"}, nil)
	in := []types.ToolSpec{
		{Name: "Bash"},
		{Name: "Read"},
		{Name: "Write"},
	}
	out := FilterToolsByDenyRules(in, ctx)
	if len(out) != 1 || out[0].Name != "Read" {
		t.Fatalf("got %#v", out)
	}
}

func TestDenyMcpServerPrefix(t *testing.T) {
	t.Parallel()
	ctx := makeContext([]string{"mcp__github"}, nil)
	tool := types.ToolSpec{
		Name:    "ignored",
		MCPInfo: &types.MCPInfo{ServerName: "github", ToolName: "list_issues"},
	}
	if GetDenyRuleForTool(ctx, tool) == nil {
		t.Fatal("expected server-level deny")
	}
}

func TestDenyMcpExactTool(t *testing.T) {
	t.Parallel()
	ctx := makeContext([]string{"mcp__github__list_issues"}, nil)
	tool := types.ToolSpec{
		Name:    "list_issues",
		MCPInfo: &types.MCPInfo{ServerName: "github", ToolName: "list_issues"},
	}
	if GetDenyRuleForTool(ctx, tool) == nil {
		t.Fatal("expected exact mcp deny")
	}
}

func TestLegacyTaskAliasInRule(t *testing.T) {
	t.Parallel()
	ctx := makeContext([]string{"Task"}, nil)
	// Rule "Task" normalizes to toolName "Agent" in PermissionRuleValueFromString path — but wait,
	// NormalizeLegacyToolName is applied to full string when no parens: PermissionRuleValueFromString("Task") -> ToolName Agent.
	r := GetDenyRuleForTool(ctx, types.ToolSpec{Name: "Agent"})
	if r == nil {
		t.Fatal("Task deny should match Agent tool")
	}
}
