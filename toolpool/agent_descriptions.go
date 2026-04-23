package toolpool

import (
	"goc/agents/builtin"
	"goc/commands"
)

// AgentInfo holds information about an agent for description formatting
type AgentInfo struct {
	AgentType       string
	WhenToUse       string
	Tools           []string
	DisallowedTools []string
}

// agentInfosFromBuiltins converts resolved builtins to AgentInfo for prompt generation.
// Same source as runtime (agents/builtin/agents.go + feature gates), matching TS passing
// loaded AgentDefinitions into getPrompt instead of a hardcoded subset.
func agentInfosFromBuiltins(agents []builtin.BuiltinAgent) []AgentInfo {
	out := make([]AgentInfo, len(agents))
	for i, a := range agents {
		var tools, deny []string
		if len(a.Tools) > 0 {
			tools = append([]string(nil), a.Tools...)
		}
		if len(a.DisallowedTools) > 0 {
			deny = append([]string(nil), a.DisallowedTools...)
		}
		out[i] = AgentInfo{
			AgentType:       a.AgentType,
			WhenToUse:       a.WhenToUse,
			Tools:           tools,
			DisallowedTools: deny,
		}
	}
	return out
}

// AgentToolDescription returns the full description with comprehensive usage guidance.
// Builtin rows come from GetBuiltInAgents(ConfigFromEnv(), empty GuideContext), consistent
// with runtime agent loading; prompt body is built in this package (no tools.init side effect).
func AgentToolDescription() string {
	infos := agentInfosFromBuiltins(builtin.GetBuiltInAgents(builtin.ConfigFromEnv(), builtin.GuideContext{}))
	opts := AgentPromptOptions{
		IsForkEnabled:     commands.ForkSubagentEnabled(commands.GouDemoSystemOpts{}),
		HasEmbeddedSearch: !EmbeddedSearchToolsActive(),
		IsProUser:         true,
	}
	return AgentPromptWithOptions(infos, opts)
}
