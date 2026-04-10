package messagesapi

// Mirrors src/utils/permissions/permissionRuleParser.ts LEGACY_TOOL_NAME_ALIASES (KAIROS/Brief omitted).
func normalizeLegacyToolName(name string) string {
	switch name {
	case "Task":
		return "Agent"
	case "KillShell":
		return "TaskStop"
	case "AgentOutputTool", "BashOutputTool":
		return "TaskOutput"
	default:
		return name
	}
}

func toolMatchesName(tool ToolSpec, name string) bool {
	if tool.Name == name {
		return true
	}
	for _, a := range tool.Aliases {
		if a == name {
			return true
		}
	}
	return false
}

func findToolByName(tools []ToolSpec, name string) *ToolSpec {
	for i := range tools {
		if toolMatchesName(tools[i], name) {
			return &tools[i]
		}
	}
	return nil
}
