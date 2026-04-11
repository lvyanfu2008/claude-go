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
