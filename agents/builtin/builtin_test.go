package builtin

import (
	"strings"
	"testing"
)

func TestGetBuiltInAgents_defaultIncludesGuide(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "")
	t.Setenv("CLAUDE_CODE_NONINTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "")
	t.Setenv("FEATURE_VERIFICATION_AGENT", "")
	t.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "")

	cfg := Config{Entrypoint: ""}
	agents := GetBuiltInAgents(cfg, GuideContext{})
	if len(agents) != 3 {
		t.Fatalf("got %d agents, want 3 (general, statusline, guide)", len(agents))
	}
	wantTypes := []string{"general-purpose", "statusline-setup", "claude-code-guide"}
	for i, w := range wantTypes {
		if agents[i].AgentType != w {
			t.Fatalf("agents[%d].AgentType = %q, want %q", i, agents[i].AgentType, w)
		}
	}
}

func TestGetBuiltInAgents_sdkDisablesAll(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "1")
	t.Setenv("CLAUDE_CODE_NONINTERACTIVE", "1")

	cfg := ConfigFromEnv()
	if len(GetBuiltInAgents(cfg, GuideContext{})) != 0 {
		t.Fatal("expected no built-in agents when SDK disable + noninteractive")
	}
}

func TestGetBuiltInAgents_sdkEntrypointOmitsGuide(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "")
	t.Setenv("CLAUDE_CODE_NONINTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-ts")
	t.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "")
	t.Setenv("FEATURE_VERIFICATION_AGENT", "")

	cfg := ConfigFromEnv()
	agents := GetBuiltInAgents(cfg, GuideContext{})
	if len(agents) != 2 {
		t.Fatalf("got %d agents, want 2 without guide", len(agents))
	}
}

func TestGetBuiltInAgents_explorePlanFeature(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "")
	t.Setenv("CLAUDE_CODE_NONINTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "1")
	t.Setenv("FEATURE_VERIFICATION_AGENT", "")

	cfg := ConfigFromEnv()
	agents := GetBuiltInAgents(cfg, GuideContext{})
	if len(agents) != 5 {
		t.Fatalf("got %d agents, want 5 with explore+plan", len(agents))
	}
	if agents[2].AgentType != "Explore" || agents[3].AgentType != "Plan" {
		t.Fatalf("unexpected order/types: %#v", []string{agents[2].AgentType, agents[3].AgentType})
	}
}

func TestGetBuiltInAgents_verificationFeature(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS", "")
	t.Setenv("CLAUDE_CODE_NONINTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS", "")
	t.Setenv("FEATURE_VERIFICATION_AGENT", "1")
	t.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "1")

	cfg := ConfigFromEnv()
	agents := GetBuiltInAgents(cfg, GuideContext{})
	if len(agents) != 4 {
		t.Fatalf("got %d agents, want 4 with verification", len(agents))
	}
	last := agents[len(agents)-1]
	if last.AgentType != "verification" || !last.Background {
		t.Fatalf("last agent: %#v", last)
	}
	if !strings.Contains(last.SystemPrompt, "VERDICT:") {
		t.Fatal("verification prompt should mention VERDICT")
	}
}

func TestClaudeCodeGuideSystemPrompt_3pFeedback(t *testing.T) {
	cfg := Config{Using3PServices: true, IssuesExplainer: "ExampleCorp support"}
	p := ClaudeCodeGuideSystemPrompt(cfg, GuideContext{})
	if !strings.Contains(p, "ExampleCorp support") {
		t.Fatal(p)
	}
	if strings.Contains(p, "/feedback") {
		t.Fatal("3P guide should not mention /feedback")
	}
}

func TestClaudeCodeGuideSystemPrompt_customSkillsSection(t *testing.T) {
	cfg := Config{}
	ctx := GuideContext{
		Commands: []GuideCommand{
			{Name: "foo", Description: "bar", Type: "prompt"},
		},
	}
	p := ClaudeCodeGuideSystemPrompt(cfg, ctx)
	if !strings.Contains(p, "/foo: bar") {
		t.Fatal(p)
	}
}

func TestVerificationSystemPrompt_substitutesToolNames(t *testing.T) {
	p := VerificationSystemPrompt()
	if !strings.Contains(p, ToolBash) || !strings.Contains(p, ToolWebFetch) {
		t.Fatal(p)
	}
}
