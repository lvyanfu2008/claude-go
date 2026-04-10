package permissionrules

import "strings"

// PermissionRuleValue mirrors PermissionRuleValue in src/types/permissions.ts / PermissionRule.js.
// RuleContent nil means undefined (whole-tool rule in toolMatchesRule).
type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

// Legacy tool name aliases (src/utils/permissions/permissionRuleParser.ts LEGACY_TOOL_NAME_ALIASES).
// KAIROS Brief entry is omitted unless we add a build tag later.
var legacyToolNameAliases = map[string]string{
	"Task":            "Agent",
	"KillShell":       "TaskStop",
	"AgentOutputTool": "TaskOutput",
	"BashOutputTool":  "TaskOutput",
}

// NormalizeLegacyToolName mirrors normalizeLegacyToolName in permissionRuleParser.ts.
func NormalizeLegacyToolName(name string) string {
	if c, ok := legacyToolNameAliases[name]; ok {
		return c
	}
	return name
}

// UnescapeRuleContent mirrors unescapeRuleContent in permissionRuleParser.ts.
func UnescapeRuleContent(content string) string {
	s := strings.ReplaceAll(content, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// PermissionRuleValueFromString mirrors permissionRuleValueFromString in permissionRuleParser.ts.
func PermissionRuleValueFromString(ruleString string) PermissionRuleValue {
	openIdx := findFirstUnescapedChar(ruleString, '(')
	if openIdx < 0 {
		return PermissionRuleValue{ToolName: NormalizeLegacyToolName(ruleString)}
	}
	closeIdx := findLastUnescapedChar(ruleString, ')')
	if closeIdx <= openIdx || closeIdx != len(ruleString)-1 {
		return PermissionRuleValue{ToolName: NormalizeLegacyToolName(ruleString)}
	}
	toolName := ruleString[:openIdx]
	rawContent := ruleString[openIdx+1 : closeIdx]
	if toolName == "" {
		return PermissionRuleValue{ToolName: NormalizeLegacyToolName(ruleString)}
	}
	if rawContent == "" || rawContent == "*" {
		return PermissionRuleValue{ToolName: NormalizeLegacyToolName(toolName)}
	}
	content := UnescapeRuleContent(rawContent)
	return PermissionRuleValue{
		ToolName:    NormalizeLegacyToolName(toolName),
		RuleContent: &content,
	}
}

func findFirstUnescapedChar(str string, char byte) int {
	for i := 0; i < len(str); i++ {
		if str[i] != char {
			continue
		}
		backslashCount := 0
		for j := i - 1; j >= 0 && str[j] == '\\'; j-- {
			backslashCount++
		}
		if backslashCount%2 == 0 {
			return i
		}
	}
	return -1
}

func findLastUnescapedChar(str string, char byte) int {
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] != char {
			continue
		}
		backslashCount := 0
		for j := i - 1; j >= 0 && str[j] == '\\'; j-- {
			backslashCount++
		}
		if backslashCount%2 == 0 {
			return i
		}
	}
	return -1
}
