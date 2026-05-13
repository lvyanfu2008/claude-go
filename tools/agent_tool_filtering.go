package tools

// ALL_AGENT_DISALLOWED_TOOLS lists tools that are globally denied for all subagents.
// Mirrors TS ALL_AGENT_DISALLOWED_TOOLS in src/constants/tools.ts.
// Agent is disallowed to prevent recursive subagent spawning.
var ALL_AGENT_DISALLOWED_TOOLS = []string{"Agent"}

// FilterToolsForAgent removes globally disallowed tools from an allowed-tools list.
func FilterToolsForAgent(allowed []string) []string {
	if len(allowed) == 0 {
		return allowed
	}
	disallowed := make(map[string]struct{}, len(ALL_AGENT_DISALLOWED_TOOLS))
	for _, t := range ALL_AGENT_DISALLOWED_TOOLS {
		disallowed[t] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for _, t := range allowed {
		if _, blocked := disallowed[t]; !blocked {
			out = append(out, t)
		}
	}
	return out
}
