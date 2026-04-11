package toolpool

import (
	"os"
	"strings"

	"goc/commands"
	"goc/commands/featuregates"
	"goc/types"
)

// GoAllowedChannelsConfigured mirrors TS getAllowedChannels().length > 0 for hosts
// without bootstrap state: set CLAUDE_CODE_GO_ALLOWED_CHANNELS to a non-empty
// comma-separated list (e.g. "discord" or "slack,discord") when channel relay is active.
func GoAllowedChannelsConfigured() bool {
	s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_GO_ALLOWED_CHANNELS"))
	if s == "" {
		return false
	}
	for _, part := range strings.Split(s, ",") {
		if strings.TrimSpace(part) != "" {
			return true
		}
	}
	return false
}

// AskUserQuestionToolEnabled mirrors AskUserQuestionTool.isEnabled in
// src/tools/AskUserQuestionTool/AskUserQuestionTool.tsx.
func AskUserQuestionToolEnabled() bool {
	if !featuregates.Feature("KAIROS") && !featuregates.Feature("KAIROS_CHANNELS") {
		return true
	}
	if !GoAllowedChannelsConfigured() {
		return true
	}
	return false
}

func toolSpecPerToolEnabled(t types.ToolSpec) bool {
	switch t.Name {
	case "AskUserQuestion":
		return AskUserQuestionToolEnabled()
	case "TodoWrite":
		// Mirrors TodoWriteTool.isEnabled (TS): hidden when Todo v2 is on.
		return !commands.TodoV2Enabled()
	case "TaskCreate", "TaskGet", "TaskList", "TaskUpdate":
		return commands.TodoV2Enabled()
	default:
		return true
	}
}

// FilterToolsByPerToolEnabled mirrors the final isEnabled() pass in src/tools.ts getTools (lines 323–324).
func FilterToolsByPerToolEnabled(tools []types.ToolSpec) []types.ToolSpec {
	if len(tools) == 0 {
		return tools
	}
	out := make([]types.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if toolSpecPerToolEnabled(t) {
			out = append(out, t)
		}
	}
	return out
}
