package tools

// ALL_AGENT_DISALLOWED_TOOLS lists tools that are globally denied for all subagents.
// Mirrors TS ALL_AGENT_DISALLOWED_TOOLS in src/constants/tools.ts.
// These tools are denied because they would allow a subagent to:
//   - View internal task/agent output (TaskOutput)
//   - Enter or exit plan mode, changing the parent's execution state (EnterPlanMode, ExitPlanMode)
//   - Ask the user questions, breaking the subagent boundary (AskUserQuestion)
//   - Kill tasks that may belong to the parent or other agents (TaskStop)
//   - Spawn further subagents recursively (Agent)
var ALL_AGENT_DISALLOWED_TOOLS = []string{
	"TaskOutput",
	"ExitPlanMode",
	"EnterPlanMode",
	"AskUserQuestion",
	"TaskStop",
	"Agent",
}

// CUSTOM_AGENT_DISALLOWED_TOOLS is the superset used for non-built-in agents.
// Same as ALL_AGENT_DISALLOWED_TOOLS in practice (TS CUSTOM_AGENT_DISALLOWED_TOOLS).
var CUSTOM_AGENT_DISALLOWED_TOOLS = ALL_AGENT_DISALLOWED_TOOLS

// ASYNC_AGENT_ALLOWED_TOOLS is the positive allowlist for async/background agents.
// Async agents can only use tools in this list, mirroring TS ASYNC_AGENT_ALLOWED_TOOLS.
var ASYNC_AGENT_ALLOWED_TOOLS = map[string]struct{}{
	"Read":              {},
	"Write":             {},
	"Edit":              {},
	"Glob":              {},
	"Grep":              {},
	"WebSearch":         {},
	"WebFetch":          {},
	"Bash":              {},
	"NotebookEdit":      {},
	"Skill":             {},
	"StructuredOutput":  {},
	"ToolSearch":        {},
	"TaskCreate":        {},
	"TaskGet":           {},
	"TaskList":          {},
	"TaskUpdate":        {},
	"EnterWorktree":     {},
	"ExitWorktree":      {},
	"CronCreate":        {},
	"CronDelete":        {},
	"CronList":          {},
	"SendMessage":       {},
	"TodoWrite":         {},
}

// FilterToolsForAgent removes globally disallowed tools from an allowed-tools list.
func FilterToolsForAgent(allowed []string) []string {
	return filterToolsByDisallowed(allowed, ALL_AGENT_DISALLOWED_TOOLS)
}

// FilterToolsForCustomAgent filters tools for non-built-in agents using the custom disallowed set.
func FilterToolsForCustomAgent(allowed []string) []string {
	return filterToolsByDisallowed(allowed, CUSTOM_AGENT_DISALLOWED_TOOLS)
}

// FilterToolsForAsyncAgent restricts the allowed tools to the async-agent allowlist.
// When the agent runs in background mode, only tools in ASYNC_AGENT_ALLOWED_TOOLS are permitted.
func FilterToolsForAsyncAgent(allowed []string) []string {
	if len(allowed) == 0 {
		return allowed
	}
	out := make([]string, 0, len(allowed))
	for _, t := range allowed {
		if _, ok := ASYNC_AGENT_ALLOWED_TOOLS[t]; ok {
			out = append(out, t)
		}
	}
	return out
}

// resolveAgentTools applies agent tool filtering including async allowlist when running in background.
func resolveAgentTools(def AgentDefinition, runInBackground bool, available []string) []string {
	result := ResolveAllowedTools(def, available)
	if runInBackground || def.Background {
		result = FilterToolsForAsyncAgent(result)
	}
	return result
}

func filterToolsByDisallowed(allowed, disallowed []string) []string {
	if len(allowed) == 0 {
		return allowed
	}
	deny := make(map[string]struct{}, len(disallowed))
	for _, t := range disallowed {
		deny[t] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for _, t := range allowed {
		if _, blocked := deny[t]; !blocked {
			out = append(out, t)
		}
	}
	return out
}
