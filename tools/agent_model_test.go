package tools

import (
	"os"
	"testing"
)

func TestGetAgentModel_InheritUsesParentModel(t *testing.T) {
	result := GetAgentModel("inherit", "claude-opus-4-20250514", "")
	if result != "claude-opus-4-20250514" {
		t.Errorf("inherit should return parent model, got %q", result)
	}
}

func TestGetAgentModel_ToolSpecifiedModelWins(t *testing.T) {
	result := GetAgentModel("inherit", "claude-opus-4-20250514", "sonnet")
	if result == "claude-opus-4-20250514" {
		t.Error("tool-specified sonnet should not return parent opus")
	}
}

func TestGetAgentModel_EmptyAgentModelDefaultsToInherit(t *testing.T) {
	result := GetAgentModel("", "claude-sonnet-4-20250514", "")
	if result != "claude-sonnet-4-20250514" {
		t.Errorf("empty agent model should default to inherit, got %q", result)
	}
}

func TestGetAgentModel_AliasMatchesParentTier(t *testing.T) {
	result := GetAgentModel("", "claude-sonnet-4-20250514", "sonnet")
	if result != "claude-sonnet-4-20250514" {
		t.Errorf("sonnet tool + sonnet parent should return parent model, got %q", result)
	}
}

func TestGetAgentModel_EnvVarOverrides(t *testing.T) {
	os.Setenv("CLAUDE_CODE_SUBAGENT_MODEL", "claude-haiku-4-5-20251001")
	defer os.Unsetenv("CLAUDE_CODE_SUBAGENT_MODEL")
	result := GetAgentModel("sonnet", "claude-opus-4-20250514", "")
	if result != "claude-haiku-4-5-20251001" {
		t.Errorf("env var should override everything, got %q", result)
	}
}

func TestGetAgentModel_EnvVarWithAlias(t *testing.T) {
	os.Setenv("CLAUDE_CODE_SUBAGENT_MODEL", "haiku")
	defer os.Unsetenv("CLAUDE_CODE_SUBAGENT_MODEL")
	result := GetAgentModel("sonnet", "claude-opus-4-20250514", "")
	if result == "" || result == "haiku" {
		t.Errorf("env var alias should resolve to a concrete model ID, got %q", result)
	}
}

func TestGetAgentModel_ToolModelWinsOverAgentDef(t *testing.T) {
	result := GetAgentModel("haiku", "claude-opus-4-20250514", "sonnet")
	// tool says sonnet, agent def says haiku, parent is opus — sonnet wins
	if result == "claude-opus-4-20250514" {
		t.Error("tool-specified model should win over agent definition")
	}
}

func TestGetAgentModel_AgentDefModelWinsWhenNoToolModel(t *testing.T) {
	result := GetAgentModel("haiku", "claude-opus-4-20250514", "")
	// agent def says haiku, parent is opus — haiku tier doesn't match parent
	if result == "claude-opus-4-20250514" {
		t.Error("agent def haiku should not match parent opus tier, should resolve separately")
	}
}

func TestGetAgentModel_AgentDefSonnetMatchesParentSonnet(t *testing.T) {
	result := GetAgentModel("sonnet", "claude-sonnet-4-20250514", "")
	if result != "claude-sonnet-4-20250514" {
		t.Errorf("agent def sonnet + parent sonnet should return parent model, got %q", result)
	}
}

func TestGetAgentModel_PassthroughConcreteModel(t *testing.T) {
	result := GetAgentModel("gpt-4o", "claude-sonnet-4-20250514", "")
	if result != "gpt-4o" {
		t.Errorf("concrete model ID should pass through, got %q", result)
	}
}

func TestAliasMatchesParentTier(t *testing.T) {
	tests := []struct {
		alias       string
		parentModel string
		want        bool
	}{
		{"sonnet", "claude-sonnet-4-20250514", true},
		{"sonnet", "claude-sonnet-4-5-20250514", true},
		{"sonnet", "claude-opus-4-20250514", false},
		{"sonnet", "claude-haiku-4-5-20251001", false},
		{"opus", "claude-opus-4-20250514", true},
		{"opus", "claude-opus-4-6-20250514", true},
		{"opus", "claude-sonnet-4-20250514", false},
		{"haiku", "claude-haiku-4-5-20251001", true},
		{"haiku", "claude-sonnet-4-20250514", false},
		{"inherit", "anything", false},
		{"", "anything", false},
		{"gpt-4o", "anything", false},
	}
	for _, tt := range tests {
		got := aliasMatchesParentTier(tt.alias, tt.parentModel)
		if got != tt.want {
			t.Errorf("aliasMatchesParentTier(%q, %q) = %v, want %v",
				tt.alias, tt.parentModel, got, tt.want)
		}
	}
}

func TestResolveAliasToModel_Passthrough(t *testing.T) {
	result := resolveAliasToModel("claude-sonnet-4-20250514")
	if result != "claude-sonnet-4-20250514" {
		t.Errorf("concrete model should pass through, got %q", result)
	}
}

func TestResolveAliasToModel_Empty(t *testing.T) {
	result := resolveAliasToModel("")
	if result == "" {
		t.Error("empty should return default model")
	}
}

func TestResolveAliasToModel_FamilyAliases(t *testing.T) {
	for _, alias := range []string{"sonnet", "opus", "haiku"} {
		result := resolveAliasToModel(alias)
		if result == alias {
			t.Errorf("alias %q should be resolved to concrete model, got same string back", alias)
		}
		if result == "" {
			t.Errorf("alias %q should resolve to non-empty model", alias)
		}
	}
}
