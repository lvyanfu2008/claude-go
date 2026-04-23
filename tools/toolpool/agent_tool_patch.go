package toolpool

import (
	"strings"

	"goc/agents/builtin"
	"goc/types"
)

const agentListingAnchor = "Available agent types and the tools they have access to:"
const agentListingTail = "When using the Agent tool"

// PatchAgentToolDescriptionWithBuiltins fills the empty agent list in the embedded tools_api.json
// Agent.description (export often omits dynamic lines). Mirrors TS getPrompt when
// shouldInjectAgentListInMessages() is false — formatAgentLine for each built-in agent.
func PatchAgentToolDescriptionWithBuiltins(specs []types.ToolSpec, agents []builtin.BuiltinAgent) []types.ToolSpec {
	if len(agents) == 0 {
		return specs
	}
	out := append([]types.ToolSpec(nil), specs...)
	var lines []string
	for i := range agents {
		lines = append(lines, builtin.FormatAgentLine(agents[i]))
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
