package permissionrules

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func freshCtx() *types.ToolPermissionContextData {
	ctx := types.EmptyToolPermissionContextData()
	return &ctx
}

func rulesJSON(m map[string][]string) json.RawMessage {
	b, _ := json.Marshal(m)
	return json.RawMessage(b)
}

func TestApplyPermissionUpdate_AddRules(t *testing.T) {
	ctx := freshCtx()

	ApplyPermissionUpdate(ctx, PermissionUpdate{
		Type:        "addRules",
		Destination: "userSettings",
		Behavior:    "deny",
		Rules:       []string{"Agent(Explore)", "Bash(rm -rf *)"},
	})

	denyMap := parseRulesMap(ctx.AlwaysDenyRules)
	if len(denyMap["userSettings"]) != 2 {
		t.Fatalf("expected 2 deny rules, got %d", len(denyMap["userSettings"]))
	}

	// Adding duplicate should not double-count.
	ApplyPermissionUpdate(ctx, PermissionUpdate{
		Type:        "addRules",
		Destination: "userSettings",
		Behavior:    "deny",
		Rules:       []string{"Agent(Explore)"},
	})
	denyMap = parseRulesMap(ctx.AlwaysDenyRules)
	if len(denyMap["userSettings"]) != 2 {
		t.Fatalf("expected still 2 deny rules after duplicate add, got %d", len(denyMap["userSettings"]))
	}
}

func TestApplyPermissionUpdate_ReplaceRules(t *testing.T) {
	ctx := freshCtx()
	ctx.AlwaysDenyRules = rulesJSON(map[string][]string{
		"userSettings": {"Agent(Explore)", "Agent(Plan)"},
	})

	ApplyPermissionUpdate(ctx, PermissionUpdate{
		Type:        "replaceRules",
		Destination: "userSettings",
		Behavior:    "deny",
		Rules:       []string{"Bash(rm -rf *)"},
	})

	denyMap := parseRulesMap(ctx.AlwaysDenyRules)
	if len(denyMap["userSettings"]) != 1 {
		t.Fatalf("expected 1 deny rule after replace, got %d", len(denyMap["userSettings"]))
	}
	if denyMap["userSettings"][0] != "Bash(rm -rf *)" {
		t.Fatalf("unexpected rule: %s", denyMap["userSettings"][0])
	}
}

func TestApplyPermissionUpdate_RemoveRules(t *testing.T) {
	ctx := freshCtx()
	ctx.AlwaysDenyRules = rulesJSON(map[string][]string{
		"userSettings": {"Agent(Explore)", "Agent(Plan)", "Bash(rm -rf *)"},
	})

	ApplyPermissionUpdate(ctx, PermissionUpdate{
		Type:        "removeRules",
		Destination: "userSettings",
		Behavior:    "deny",
		Rules:       []string{"Agent(Explore)"},
	})

	denyMap := parseRulesMap(ctx.AlwaysDenyRules)
	if len(denyMap["userSettings"]) != 2 {
		t.Fatalf("expected 2 deny rules after remove, got %d", len(denyMap["userSettings"]))
	}
}

func TestApplyPermissionUpdate_SetMode(t *testing.T) {
	ctx := freshCtx()

	ApplyPermissionUpdate(ctx, PermissionUpdate{
		Type: "setMode",
		Mode: "auto",
	})

	if ctx.Mode != "auto" {
		t.Fatalf("expected mode 'auto', got %q", ctx.Mode)
	}
}

func TestApplyPermissionUpdates_Batch(t *testing.T) {
	ctx := freshCtx()

	ApplyPermissionUpdates(ctx, []PermissionUpdate{
		{Type: "setMode", Mode: "auto"},
		{Type: "addRules", Destination: "session", Behavior: "deny", Rules: []string{"Agent(Explore)"}},
		{Type: "addRules", Destination: "session", Behavior: "allow", Rules: []string{"Read"}},
	})

	if ctx.Mode != "auto" {
		t.Fatalf("expected mode 'auto', got %q", ctx.Mode)
	}
	denyMap := parseRulesMap(ctx.AlwaysDenyRules)
	if len(denyMap["session"]) != 1 {
		t.Fatalf("expected 1 deny rule, got %d", len(denyMap["session"]))
	}
	allowMap := parseRulesMap(ctx.AlwaysAllowRules)
	if len(allowMap["session"]) != 1 {
		t.Fatalf("expected 1 allow rule, got %d", len(allowMap["session"]))
	}
}
