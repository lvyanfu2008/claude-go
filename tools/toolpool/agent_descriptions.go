package toolpool

import (
	"goc/agents/builtin"
	"goc/commands"
	"goc/permissionrules"
	"goc/types"
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
// It is [AgentToolDescriptionWithPermission] with an empty [types.ToolPermissionContextData].
// Builtin rows are filtered the same way as in TS (filterDeniedAgents) when permission is non-empty
// via [AgentToolDescriptionWithPermission].
func AgentToolDescription() string {
	return AgentToolDescriptionWithPermission(types.EmptyToolPermissionContextData())
}

// AgentToolDescriptionWithPermission returns the native Agent tool description, omitting
// subagent types that alwaysDeny (e.g. Agent(Explore)) from the list — mirrors filterDeniedAgents in TS.
func AgentToolDescriptionWithPermission(perm types.ToolPermissionContextData) string {
	infos := agentInfosFromBuiltins(builtin.GetBuiltInAgents(builtin.ConfigFromEnv(), builtin.GuideContext{}))
	infos = permissionrules.FilterDeniedAgents(infos, func(a AgentInfo) string { return a.AgentType }, perm, "Agent")
	opts := AgentPromptOptions{
		IsForkEnabled:     commands.ForkSubagentEnabled(commands.GouDemoSystemOpts{}),
		HasEmbeddedSearch: !EmbeddedSearchToolsActive(),
		IsProUser:         true,
	}
	return AgentPromptWithOptions(infos, opts)
}
