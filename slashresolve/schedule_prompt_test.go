package slashresolve

import (
	"strings"
	"testing"
)

func TestBuildSchedulePrompt_containsToolNames(t *testing.T) {
	p := buildSchedulePrompt(SchedulePromptOpts{
		UserTimezone:     "UTC",
		ConnectorsInfo:   "none",
		EnvironmentsInfo: "Available environments:\n- default (id: e1, kind: anthropic_cloud)",
		UserArgs:         "",
	})
	if !strings.Contains(p, remoteTriggerToolName) || !strings.Contains(p, askUserQuestionToolName) {
		t.Fatal(p[:400])
	}
}

func TestResolveSchedule_noOAuthMessage(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	res, err := resolveSchedule("", &BundledResolveOptions{Cwd: home})
	if err != nil {
		t.Fatal(err)
	}
	if res.UserText != scheduleAuthRequiredMsg {
		t.Fatalf("got %q", res.UserText)
	}
}
