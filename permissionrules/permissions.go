package permissionrules

import (
	"encoding/json"
	"strings"

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

// GetDenyRuleForAgent mirrors getDenyRuleForAgent in src/utils/permissions/permissions.ts.
// For example, alwaysDeny "Agent(Explore)" denies the Explore subagent. Compares agent types
// with [strings.EqualFold] so "explore" matches "Explore".
func GetDenyRuleForAgent(
	ctx types.ToolPermissionContextData,
	agentToolName string,
	agentType string,
) *PermissionRule {
	agentToolName = NormalizeLegacyToolName(agentToolName)
	for _, rule := range GetDenyRules(ctx) {
		if rule.RuleValue.ToolName != agentToolName {
			continue
		}
		if rule.RuleValue.RuleContent == nil {
			continue
		}
		if strings.EqualFold(*rule.RuleValue.RuleContent, agentType) {
			r := rule
			return &r
		}
	}
	return nil
}

// FilterDeniedAgents mirrors filterDeniedAgents in src/utils/permissions/permissions.ts
// (agents with agentType denied via Agent(AgentName) in alwaysDeny are removed).
func FilterDeniedAgents[T any](
	agents []T,
	getAgentType func(T) string,
	ctx types.ToolPermissionContextData,
	agentToolName string,
) []T {
	agentToolName = NormalizeLegacyToolName(agentToolName)
	denied := make(map[string]struct{})
	for _, rule := range GetDenyRules(ctx) {
		if rule.RuleValue.ToolName != agentToolName {
			continue
		}
		if rule.RuleValue.RuleContent == nil {
			continue
		}
		denied[*rule.RuleValue.RuleContent] = struct{}{}
	}
	if len(denied) == 0 {
		return append([]T(nil), agents...)
	}
	out := make([]T, 0, len(agents))
	for _, a := range agents {
		t := getAgentType(a)
		if isAgentTypeDeniedSet(denied, t) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func isAgentTypeDeniedSet(denied map[string]struct{}, agentType string) bool {
	if agentType == "" {
		return false
	}
	for k := range denied {
		if strings.EqualFold(k, agentType) {
			return true
		}
	}
	return false
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
