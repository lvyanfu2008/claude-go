package messagesapi

import "goc/permissionrules"

// Delegates to [permissionrules.NormalizeLegacyToolName] (same map as permissionRuleParser.ts LEGACY_TOOL_NAME_ALIASES).
func normalizeLegacyToolName(name string) string {
	return permissionrules.NormalizeLegacyToolName(name)
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
