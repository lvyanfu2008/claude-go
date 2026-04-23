package deferredtoolsdelta

import (
	"strings"
	"testing"
)

func TestToolSearchToolDescriptionUsesSystemReminderWhenDeltaEnabled(t *testing.T) {
	t.Setenv("USER_TYPE", "ant")
	t.Setenv("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA", "")
	got := ToolSearchToolDescription()
	if !strings.Contains(got, "<system-reminder>") {
		t.Fatalf("expected system-reminder hint for USER_TYPE=ant, got:\n%s", got)
	}
	if strings.Contains(got, "<available-deferred-tools>") {
		t.Fatalf("did not expect available-deferred-tools when delta path on")
	}
	if !strings.Contains(got, "select:Read,Edit,Grep") {
		t.Fatalf("expected TS query-form examples in description")
	}
}

func TestToolSearchToolDescriptionUsesAvailableDeferredWhenDeltaOff(t *testing.T) {
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA", "0")
	t.Setenv("CLAUDE_CODE_TENGU_GLACIER_2XR", "")
	t.Setenv("CLAUDE_CODE_TENGU_TENGU_GLACIER_2XR", "")
	got := ToolSearchToolDescription()
	if !strings.Contains(got, "<available-deferred-tools>") {
		t.Fatalf("expected available-deferred-tools hint when delta off, got:\n%s", got)
	}
	if strings.Contains(got, "<system-reminder> messages.") {
		t.Fatalf("did not expect system-reminder deferred hint when delta off")
	}
}

func TestToolSearchToolDescriptionGlacierEnvUsesSystemReminderHint(t *testing.T) {
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA", "0")
	t.Setenv("CLAUDE_CODE_TENGU_GLACIER_2XR", "1")
	t.Setenv("CLAUDE_CODE_TENGU_TENGU_GLACIER_2XR", "")
	got := ToolSearchToolDescription()
	if !strings.Contains(got, "<system-reminder>") {
		t.Fatalf("expected system-reminder hint when glacier env on, got:\n%s", got)
	}
}
