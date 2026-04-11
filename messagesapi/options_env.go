package messagesapi

import (
	"os"
	"strconv"
	"strings"

	"goc/commands/featuregates"
)

func envTruthyMsg(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envDefinedFalsy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// PlanModeInterviewPhaseFromEnv mirrors src/utils/planModeV2.ts isPlanModeInterviewPhaseEnabled (GrowthBook default false → opt-in via env).
func PlanModeInterviewPhaseFromEnv() bool {
	if featuregates.UserTypeAnt() {
		return true
	}
	k := "CLAUDE_CODE_PLAN_MODE_INTERVIEW_PHASE"
	if envTruthyMsg(k) {
		return true
	}
	if envDefinedFalsy(k) {
		return false
	}
	return envTruthyMsg("CLAUDE_CODE_GO_PLAN_MODE_INTERVIEW_PHASE")
}

func parsePlanCountEnv(k string, fallback int) int {
	s := strings.TrimSpace(os.Getenv(k))
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 10 {
		return fallback
	}
	return n
}

// OptionsFromEnv builds Options using the same environment keys as the TS normalize / plan-mode path where applicable.
func OptionsFromEnv() Options {
	o := DefaultOptions()
	o.PlanModeInterviewPhase = PlanModeInterviewPhaseFromEnv()
	o.PlanPhase4Variant = strings.TrimSpace(os.Getenv("CLAUDE_CODE_GO_PEWTER_LEDGER"))
	o.PlanModeV2AgentCount = parsePlanCountEnv("CLAUDE_CODE_PLAN_V2_AGENT_COUNT", 0)
	o.PlanModeV2ExploreAgentCount = parsePlanCountEnv("CLAUDE_CODE_PLAN_V2_EXPLORE_AGENT_COUNT", 0)
	o.PlanModeEmbeddedSearchTools = featuregates.Feature("CHICAGO_MCP") || envTruthyMsg("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS")
	if featuregates.Feature("BUILTIN_EXPLORE_PLAN_AGENTS") {
		if v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_GO_EXPLORE_PLAN_AGENTS"))); v != "0" && v != "false" {
			o.ExplorePlanAgentsEnabled = true
		}
	}
	o.ExperimentalSkillSearch = featuregates.Feature("EXPERIMENTAL_SKILL_SEARCH") || strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME")) != ""
	o.VerifyPlanToolEnabled = envTruthyMsg("CLAUDE_CODE_VERIFY_PLAN")
	o.ToolSearchEnabled = envTruthyMsg("CLAUDE_CODE_GO_TOOL_SEARCH")
	return o
}
