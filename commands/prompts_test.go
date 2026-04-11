package commands

import (
	"strings"
	"testing"
)

func TestPrependBullets_matchesTS(t *testing.T) {
	got := strings.Join(PrependBullets(
		"alpha",
		[]string{"b1", "b2"},
	), "\n")
	want := " - alpha\n  - b1\n  - b2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestComputeSimpleEnvInfo_containsModelFamilyLine(t *testing.T) {
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	s := ComputeSimpleEnvInfo(SimpleEnvInfoInput{
		ModelID:                 "claude-sonnet-4-20250514",
		PrimaryWorkingDirectory: "/tmp/x",
		IsGitRepo:               false,
		PlatformGOOS:            "linux",
	})
	if !strings.Contains(s, "Claude 4.5/4.6") || !strings.Contains(s, "# Environment") {
		t.Fatal(s[:min(200, len(s))])
	}
}

func TestComputeSimpleEnvInfo_envReportModelID_likeTSDeepseek(t *testing.T) {
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	s := ComputeSimpleEnvInfo(SimpleEnvInfoInput{
		ModelID:                 "claude-sonnet-4-20250514",
		EnvReportModelID:        "deepseek-chat",
		PrimaryWorkingDirectory: "/tmp/x",
		IsGitRepo:               false,
		PlatformGOOS:            "linux",
	})
	if !strings.Contains(s, "You are powered by the model deepseek-chat.") {
		t.Fatal(s)
	}
	// TS marketing line uses "model named … The exact model ID is …" — must not appear for deepseek.
	if strings.Contains(s, "You are powered by the model named") || strings.Contains(s, "The exact model ID is claude-sonnet") {
		t.Fatal(s)
	}
	if strings.Contains(s, "Assistant knowledge cutoff is") {
		t.Fatal("non-Claude report id should not get Anthropic cutoff line")
	}
}

func TestShouldUseGlobalCacheScope_respectsDisableBetas(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	if ShouldUseGlobalCacheScope() {
		t.Fatal("expected false when experimental betas disabled")
	}
}
