package toolpool

// Coordinator allowed tool names — mirrors COORDINATOR_MODE_ALLOWED_TOOLS in src/constants/tools.ts (lines 105–110).
// AGENT_TOOL_NAME, TASK_STOP_TOOL_NAME, SEND_MESSAGE_TOOL_NAME, SYNTHETIC_OUTPUT_TOOL_NAME.
var coordinatorModeAllowedTools = map[string]struct{}{
	"Agent":            {},
	"TaskStop":         {},
	"SendMessage":      {},
	"StructuredOutput": {},
}

// prActivityToolSuffixes mirrors PR_ACTIVITY_TOOL_SUFFIXES in src/utils/toolPool.ts (lines 11–14).
var prActivityToolSuffixes = []string{
	"subscribe_pr_activity",
	"unsubscribe_pr_activity",
}
