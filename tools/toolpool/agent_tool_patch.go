package toolpool

import (
	"strings"

	"goc/agents/builtin"
	"goc/permissionrules"
	"goc/types"
)

const agentListingAnchor = "Available agent types and the tools they have access to:"
const agentListingTail = "When using the Agent tool"

// PatchAgentToolDescriptionWithBuiltins is [PatchAgentToolDescriptionWithPermission] with an empty
// permission context (no per-agent alwaysDeny filtering).
func PatchAgentToolDescriptionWithBuiltins(specs []types.ToolSpec, agents []builtin.BuiltinAgent) []types.ToolSpec {
	return PatchAgentToolDescriptionWithPermission(specs, agents, types.EmptyToolPermissionContextData())
}

// PatchAgentToolDescriptionWithPermission fills the empty agent list in the embedded tools_api.json
// Agent.description. Built-in rows are filtered with [permissionrules.FilterDeniedAgents] (TS filterDeniedAgents)
// before [builtin.FormatAgentLine] injection.
func PatchAgentToolDescriptionWithPermission(
	specs []types.ToolSpec,
	agents []builtin.BuiltinAgent,
	perm types.ToolPermissionContextData,
) []types.ToolSpec {
	if len(agents) == 0 {
		return specs
	}
	filtered := permissionrules.FilterDeniedAgents(agents, func(a builtin.BuiltinAgent) string { return a.AgentType }, perm, "Agent")
	if len(filtered) == 0 {
		return specs
	}
	out := append([]types.ToolSpec(nil), specs...)
	var lines []string
	for i := range filtered {
		lines = append(lines, builtin.FormatAgentLine(filtered[i]))
	}
	block := strings.Join(lines, "\n")
	for i := range out {
		if out[i].Name != "Agent" {
			continue
		}
		d := out[i].Description
		if !strings.Contains(d, agentListingAnchor) || !strings.Contains(d, agentListingTail) {
			continue
		}
		if strings.Contains(d, "- general-purpose:") {
			continue
		}
		old := agentListingAnchor + "\n\n\n" + agentListingTail
		if strings.Contains(d, old) {
			out[i].Description = strings.Replace(d, old, agentListingAnchor+"\n\n"+block+"\n\n"+agentListingTail, 1)
			continue
		}
		old2 := agentListingAnchor + "\n\n" + agentListingTail
		if strings.Contains(d, old2) {
			out[i].Description = strings.Replace(d, old2, agentListingAnchor+"\n\n"+block+"\n\n"+agentListingTail, 1)
		}
	}
	return out
}
