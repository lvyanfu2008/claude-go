package tools

import (
	"fmt"
	"strings"
)

const AgentToolName = "Agent"
const LegacyAgentToolName = "Task"

func FormatAgentLine(agent AgentDefinition) string {
	toolsDesc := "All tools"
	switch {
	case len(agent.Tools) > 0 && len(agent.DisallowedTools) > 0:
		effective := make([]string, 0, len(agent.Tools))
		deny := map[string]struct{}{}
		for _, d := range agent.DisallowedTools {
			deny[d] = struct{}{}
		}
		for _, t := range agent.Tools {
			if _, blocked := deny[t]; !blocked {
				effective = append(effective, t)
			}
		}
		if len(effective) == 0 {
			toolsDesc = "None"
		} else {
			toolsDesc = strings.Join(effective, ", ")
		}
	case len(agent.Tools) > 0:
		toolsDesc = strings.Join(agent.Tools, ", ")
	case len(agent.DisallowedTools) > 0:
		toolsDesc = "All tools except " + strings.Join(agent.DisallowedTools, ", ")
	}
	return fmt.Sprintf("- %s: %s (Tools: %s)", agent.AgentType, agent.WhenToUse, toolsDesc)
}

func AgentPrompt(agentDefinitions []AgentDefinition) string {
	lines := make([]string, 0, len(agentDefinitions))
	for _, a := range agentDefinitions {
		lines = append(lines, FormatAgentLine(a))
	}
	return "Launch a new agent to handle complex, multi-step tasks autonomously.\n\n" +
		"The Agent tool launches specialized agents (subprocesses) that autonomously handle complex tasks. Each agent type has specific capabilities and tools available to it.\n\n" +
		"Available agent types and the tools they have access to:\n" + strings.Join(lines, "\n") + "\n\n" +
		"When using the Agent tool, specify a subagent_type parameter to select which agent type to use. If omitted, the general-purpose agent is used."
}
