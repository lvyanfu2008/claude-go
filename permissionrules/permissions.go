package permissionrules

import (
	"encoding/json"

	"goc/types"
)

// PermissionRule mirrors rows produced by getDenyRules / getAllowRules / getAskRules (src/utils/permissions/permissions.ts).
type PermissionRule struct {
	Source       string
	RuleBehavior string // "deny" | "allow" | "ask"
	RuleValue    PermissionRuleValue
}

// PermissionRuleSources matches PERMISSION_RULE_SOURCES in src/utils/permissions/permissions.ts
// (SETTING_SOURCES + cliArg, command, session).
var PermissionRuleSources = []string{
	"userSettings",
	"projectSettings",
	"localSettings",
	"flagSettings",
	"policySettings",
	"cliArg",
	"command",
	"session",
}

func parseRulesBySource(raw json.RawMessage) map[string][]string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var m map[string][]string
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	if m == nil {
		return nil
	}
	return m
}

// GetDenyRules mirrors getDenyRules in src/utils/permissions/permissions.ts.
func GetDenyRules(ctx types.ToolPermissionContextData) []PermissionRule {
	bySource := parseRulesBySource(ctx.AlwaysDenyRules)
	var out []PermissionRule
	for _, source := range PermissionRuleSources {
		for _, s := range bySource[source] {
			out = append(out, PermissionRule{
				Source:       source,
				RuleBehavior: "deny",
				RuleValue:    PermissionRuleValueFromString(s),
			})
		}
	}
	return out
}

// GetAskRules mirrors getAskRules in src/utils/permissions/permissions.ts.
func GetAskRules(ctx types.ToolPermissionContextData) []PermissionRule {
	bySource := parseRulesBySource(ctx.AlwaysAskRules)
	var out []PermissionRule
	for _, source := range PermissionRuleSources {
		for _, s := range bySource[source] {
			out = append(out, PermissionRule{
				Source:       source,
				RuleBehavior: "ask",
				RuleValue:    PermissionRuleValueFromString(s),
			})
		}
	}
	return out
}

// ToolMatchesRule mirrors toolMatchesRule in src/utils/permissions/permissions.ts (whole-tool match only).
func ToolMatchesRule(tool types.ToolSpec, rule PermissionRule) bool {
	if rule.RuleValue.RuleContent != nil {
		return false
	}
	nameForRuleMatch := GetToolNameForPermissionCheck(tool)
	rv := rule.RuleValue
	if rv.ToolName == nameForRuleMatch {
		return true
	}
	ruleInfo := McpInfoFromString(rv.ToolName)
	toolInfo := McpInfoFromString(nameForRuleMatch)
	if ruleInfo == nil || toolInfo == nil {
		return false
	}
	ruleToolName := ""
	if ruleInfo.ToolName != nil {
		ruleToolName = *ruleInfo.ToolName
	}
	if ruleToolName != "" && ruleToolName != "*" {
		return false
	}
	return ruleInfo.ServerName == toolInfo.ServerName
}

// GetDenyRuleForTool mirrors getDenyRuleForTool in src/utils/permissions/permissions.ts.
func GetDenyRuleForTool(ctx types.ToolPermissionContextData, tool types.ToolSpec) *PermissionRule {
	for _, rule := range GetDenyRules(ctx) {
		if ToolMatchesRule(tool, rule) {
			r := rule
			return &r
		}
	}
	return nil
}

// GetAskRuleForTool mirrors getAskRuleForTool in src/utils/permissions/permissions.ts.
func GetAskRuleForTool(ctx types.ToolPermissionContextData, tool types.ToolSpec) *PermissionRule {
	for _, rule := range GetAskRules(ctx) {
		if ToolMatchesRule(tool, rule) {
			r := rule
			return &r
		}
	}
	return nil
}

// FilterToolsByDenyRules mirrors filterToolsByDenyRules in src/tools.ts.
func FilterToolsByDenyRules(tools []types.ToolSpec, ctx types.ToolPermissionContextData) []types.ToolSpec {
	if len(tools) == 0 {
		return nil
	}
	out := make([]types.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if GetDenyRuleForTool(ctx, t) == nil {
			out = append(out, t)
		}
	}
	return out
}
