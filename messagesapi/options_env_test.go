package messagesapi

import (
	"testing"
)

func TestPlanModeInterviewPhaseFromEnv(t *testing.T) {
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_PLAN_MODE_INTERVIEW_PHASE", "")
	t.Setenv("CLAUDE_CODE_GO_PLAN_MODE_INTERVIEW_PHASE", "")
	if PlanModeInterviewPhaseFromEnv() {
		t.Fatal("expected false when unset")
	}
	t.Setenv("CLAUDE_CODE_PLAN_MODE_INTERVIEW_PHASE", "1")
	if !PlanModeInterviewPhaseFromEnv() {
		t.Fatal("expected true when env truthy")
	}
	t.Setenv("CLAUDE_CODE_PLAN_MODE_INTERVIEW_PHASE", "false")
	if PlanModeInterviewPhaseFromEnv() {
		t.Fatal("expected false when explicitly false")
	}
}

func TestOptionsFromEnv_ChairSermonDefaultOn(t *testing.T) {
	t.Setenv("CLAUDE_CODE_GO_CHAIR_SERMON", "")
	if !OptionsFromEnv().ChairSermon {
		t.Fatal("expected ChairSermon true when CLAUDE_CODE_GO_CHAIR_SERMON unset")
	}
	t.Setenv("CLAUDE_CODE_GO_CHAIR_SERMON", "0")
	if OptionsFromEnv().ChairSermon {
		t.Fatal("expected ChairSermon false when CLAUDE_CODE_GO_CHAIR_SERMON=0")
	}
}
