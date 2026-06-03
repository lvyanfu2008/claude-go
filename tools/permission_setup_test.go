package tools

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestPrepareContextForPlanMode_PlainEntry(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	ctx.Mode = types.PermissionDefault

	result := prepareContextForPlanMode(&ctx)

	if result.Mode != types.PermissionDefault {
		t.Errorf("mode unchanged by prepareContextForPlanMode, got %s", result.Mode)
	}
	if result.PrePlanMode == nil {
		t.Fatal("prePlanMode should be set")
	}
	if *result.PrePlanMode != types.PermissionDefault {
		t.Errorf("prePlanMode = %s, want default", *result.PrePlanMode)
	}
}

func TestPrepareContextForPlanMode_AlreadyPlan(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	ctx.Mode = types.PermissionPlan

	result := prepareContextForPlanMode(&ctx)
	if result.PrePlanMode != nil {
		t.Error("prePlanMode should be nil when already in plan mode")
	}
}

func TestIsDangerousRule(t *testing.T) {
	tests := []struct {
		rule      string
		dangerous bool
	}{
		{"Bash(*)", true},
		{"Bash", true},
		{"bash(npm run build)", true},
		{"Agent(*)", true},
		{"PowerShell(*)", true},
		{"Read(*)", false},
		{"Bash(npm test)", true},
		{"Write(file.txt)", false},
		{"Agent(Explore: *)", true},
	}
	for _, tc := range tests {
		got := isDangerousRule(tc.rule)
		if got != tc.dangerous {
			t.Errorf("isDangerousRule(%q) = %v, want %v", tc.rule, got, tc.dangerous)
		}
	}
}

func TestStripDangerousPermissionsForAutoMode(t *testing.T) {
	allowRules := map[string][]string{
		"session":      {"Bash(*)", "Read(*)", "Agent(*)"},
		"userSettings": {"Write(*)"},
	}
	allowRaw, _ := json.Marshal(allowRules)

	ctx := types.EmptyToolPermissionContextData()
	ctx.AlwaysAllowRules = allowRaw

	result := stripDangerousPermissionsForAutoMode(&ctx)

	// Session rules should have dangerous ones stripped
	var resultAllow map[string][]string
	json.Unmarshal(result.AlwaysAllowRules, &resultAllow)

	sessionRules := resultAllow["session"]
	if len(sessionRules) != 1 || sessionRules[0] != "Read(*)" {
		t.Errorf("session rules after strip = %v, want [Read(*)]", sessionRules)
	}

	// UserSettings rules should be untouched
	if len(resultAllow["userSettings"]) != 1 || resultAllow["userSettings"][0] != "Write(*)" {
		t.Errorf("userSettings rules modified: %v", resultAllow["userSettings"])
	}

	// Stripped rules should be saved
	if len(result.StrippedDangerousRules) == 0 {
		t.Fatal("strippedDangerousRules should not be empty")
	}
}

func TestRestoreDangerousPermissions(t *testing.T) {
	allowRules := map[string][]string{
		"session": {"Read(*)"},
	}
	allowRaw, _ := json.Marshal(allowRules)

	stripped := map[string][]string{
		"alwaysAllowRules": {"Bash(*)", "Agent(*)"},
	}
	strippedRaw, _ := json.Marshal(stripped)

	ctx := types.EmptyToolPermissionContextData()
	ctx.AlwaysAllowRules = allowRaw
	ctx.StrippedDangerousRules = strippedRaw

	result := restoreDangerousPermissions(&ctx)

	var resultAllow map[string][]string
	json.Unmarshal(result.AlwaysAllowRules, &resultAllow)

	sessionRules := resultAllow["session"]
	if len(sessionRules) != 3 {
		t.Errorf("session rules after restore = %v, want 3 rules", sessionRules)
	}
	if len(result.StrippedDangerousRules) != 0 {
		t.Error("strippedDangerousRules should be cleared after restore")
	}
}
