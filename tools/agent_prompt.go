package tools

import "goc/toolpool"

// Re-export prompt constants for call sites that imported this package.
const (
	AgentToolName       = toolpool.AgentToolName
	LegacyAgentToolName = toolpool.LegacyAgentToolName
	FileReadToolName    = toolpool.FileReadToolName
	FileWriteToolName   = toolpool.FileWriteToolName
	GlobToolName        = toolpool.GlobToolName
	SendMessageToolName = toolpool.SendMessageToolName
)

// AgentPromptOptions configures the agent prompt generation (toolpool type alias).
type AgentPromptOptions = toolpool.AgentPromptOptions

func agentDefinitionToInfo(a AgentDefinition) toolpool.AgentInfo {
	return toolpool.AgentInfo{
		AgentType:       a.AgentType,
		WhenToUse:       a.WhenToUse,
		Tools:           a.Tools,
		DisallowedTools: a.DisallowedTools,
	}
}

// FormatAgentLine mirrors TS formatAgentLine for AgentDefinition inputs.
func FormatAgentLine(agent AgentDefinition) string {
	return toolpool.FormatAgentToolListingLine(agentDefinitionToInfo(agent))
}

func AgentPrompt(agentDefinitions []AgentDefinition) string {
	return AgentPromptWithOptions(agentDefinitions, AgentPromptOptions{})
}

func AgentPromptWithOptions(agentDefinitions []AgentDefinition, opts AgentPromptOptions) string {
	infos := make([]toolpool.AgentInfo, len(agentDefinitions))
	for i, a := range agentDefinitions {
		infos[i] = agentDefinitionToInfo(a)
	}
	return toolpool.AgentPromptWithOptions(infos, opts)
}
