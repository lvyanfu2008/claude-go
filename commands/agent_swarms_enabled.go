package commands

import (
	"os"
	"slices"
	"strings"
)

// AgentSwarmsEnabled mirrors isAgentSwarmsEnabled in claude-code/src/utils/agentSwarmsEnabled.ts.
// Ant builds: always true. External: CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS or argv contains --agent-teams,
// and GrowthBook killswitch tengu_amber_flint (default on) — opt out with CLAUDE_CODE_GO_TENGU_AMBER_FLINT=0|false.
func AgentSwarmsEnabled() bool {
	if strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" {
		return true
	}
	if !IsEnvTruthy("CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS") && !agentTeamsFlagInArgv() {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_GO_TENGU_AMBER_FLINT")))
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return true
}

func agentTeamsFlagInArgv() bool {
	return slices.Contains(os.Args, "--agent-teams")
}
