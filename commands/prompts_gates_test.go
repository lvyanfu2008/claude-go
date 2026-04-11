package commands

import (
	"strings"
	"testing"
)

func TestForkSubagentEnabled_requiresFeatureAndInteractive(t *testing.T) {
	t.Setenv("FEATURE_FORK_SUBAGENT", "1")
	o := GouDemoSystemOpts{NonInteractiveSession: true}
	if ForkSubagentEnabled(o) {
		t.Fatal("non-interactive should disable fork")
	}
	o.NonInteractiveSession = false
	o.CoordinatorMode = true
	if ForkSubagentEnabled(o) {
		t.Fatal("coordinator should disable fork")
	}
	o.CoordinatorMode = false
	if !ForkSubagentEnabled(o) {
		t.Fatal("expected fork enabled")
	}
}

func TestGetBriefSection_skippedWhenProactive(t *testing.T) {
	t.Setenv("FEATURE_KAIROS_BRIEF", "1")
	t.Setenv("FEATURE_KAIROS", "1")
	t.Setenv("FEATURE_PROACTIVE", "1")
	o := GouDemoSystemOpts{KairosActive: true, UserMsgOptIn: true}
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "")
	if GetBriefSection(o) == "" {
		t.Fatal("expected brief section when proactive off")
	}
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "1")
	if GetBriefSection(o) != "" {
		t.Fatal("brief section must be omitted when proactive active (TS getBriefSection)")
	}
}

func TestIsMcpInstructionsDeltaEnabled_envTruthy(t *testing.T) {
	t.Setenv("CLAUDE_CODE_MCP_INSTR_DELTA", "true")
	if !IsMcpInstructionsDeltaEnabled() {
		t.Fatal()
	}
}

func TestFunctionResultClearingSection_envGate(t *testing.T) {
	t.Setenv("FEATURE_CACHED_MICROCOMPACT", "1")
	if FunctionResultClearingSection("claude-sonnet-4") != "" {
		t.Fatal("expected empty when FRC env gates off")
	}
	t.Setenv("CLAUDE_CODE_GO_CACHED_MC_FRC_ENABLED", "1")
	t.Setenv("CLAUDE_CODE_GO_CACHED_MC_SYSTEM_PROMPT_SUGGEST_SUMMARIES", "1")
	t.Setenv("CLAUDE_CODE_GO_CACHED_MC_SUPPORTED_MODELS", "sonnet")
	s := FunctionResultClearingSection("claude-sonnet-4-20250514")
	if !strings.Contains(s, "Function Result Clearing") || !strings.Contains(s, "most recent") {
		t.Fatal(s)
	}
}

func TestBuildGouDemoSystemPrompt_proactivePath(t *testing.T) {
	t.Setenv("FEATURE_KAIROS", "1")
	t.Setenv("FEATURE_PROACTIVE", "1")
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "1")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	o := GouDemoSystemOpts{
		Cwd:                   "/tmp",
		ModelID:               "m",
		SkipPromptGitDetect:   true,
		ParityGOOS:            "linux",
		KairosActive:          true,
		UserMsgOptIn:          true,
		NonInteractiveSession: false,
	}
	s := BuildGouDemoSystemPrompt(o)
	if !strings.Contains(s, "autonomous agent") || !strings.Contains(s, "# Autonomous work") {
		t.Fatal(s[:min(300, len(s))])
	}
}
