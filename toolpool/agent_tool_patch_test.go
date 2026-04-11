package toolpool

import (
	"strings"
	"testing"

	"goc/agents/builtin"
	"goc/types"
)

func TestPatchAgentToolDescriptionWithBuiltins_fillsEmptyListing(t *testing.T) {
	embed := `Launch a new agent to handle complex, multi-step tasks autonomously.

The Agent tool launches specialized agents (subprocesses) that autonomously handle complex tasks. Each agent type has specific capabilities and tools available to it.

Available agent types and the tools they have access to:


When using the Agent tool, specify a subagent_type parameter to select which agent type to use. If omitted, the general-purpose agent is used.`
	specs := []types.ToolSpec{
		{Name: "Read", Description: "x"},
		{Name: "Agent", Description: embed},
	}
	agents := []builtin.BuiltinAgent{{
		AgentType: "general-purpose",
		WhenToUse: "When in doubt, delegate.",
		Tools:     []string{"*"},
	}}
	out := PatchAgentToolDescriptionWithBuiltins(specs, agents)
	var agentDesc string
	for _, t := range out {
		if t.Name == "Agent" {
			agentDesc = t.Description
			break
		}
	}
	if !strings.Contains(agentDesc, "- general-purpose:") || !strings.Contains(agentDesc, "When in doubt") {
		t.Fatalf("missing injected line; got:\n%s", agentDesc)
	}
	if strings.Contains(agentDesc, "Available agent types and the tools they have access to:\n\n\nWhen") {
		t.Fatal("still has empty triple-newline block")
	}
}
