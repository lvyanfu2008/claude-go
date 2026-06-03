package tools

import (
	"encoding/json"
	"strings"

	"goc/appstate"
	"goc/growthbook"
	"goc/types"
)

// dangerousRulePatterns are permission rule patterns that are considered "dangerous"
// (unrestricted Bash/Agent/PowerShell). Mirrors TS stripDangerousPermissionsForAutoMode.
var dangerousRulePatterns = []string{
	"Bash(*)", "Bash",
	"Agent(*)", "Agent",
	"PowerShell(*)", "PowerShell",
}

// prepareContextForPlanMode saves prePlanMode, handles auto mode transitions.
// Mirrors TS prepareContextForPlanMode in permissionSetup.ts.
func prepareContextForPlanMode(ctx *types.ToolPermissionContextData) *types.ToolPermissionContextData {
	currentMode := ctx.Mode
	if currentMode == types.PermissionPlan {
		return ctx
	}

	mode := currentMode
	prePlanMode := mode
	result := *ctx
	result.PrePlanMode = &prePlanMode

	if shouldPlanUseAutoMode() {
		if currentMode != types.PermissionBypassPermissions {
			appstate.GlobalAutoModeState.SetActive(true)
			return stripDangerousPermissionsForAutoMode(&result)
		}
		return &result
	}

	if currentMode == types.PermissionAuto {
		appstate.GlobalAutoModeState.SetActive(false)
		return restoreDangerousPermissions(&result)
	}

	return &result
}

// handlePlanModeTransition manages global side effects for plan mode state changes.
// Mirrors TS handlePlanModeTransition in bootstrap/state.ts.
func handlePlanModeTransition(fromMode, toMode types.PermissionMode) (needsExitAttachment bool, hasExited bool, needsAutoExit bool) {
	if toMode == types.PermissionPlan && fromMode != types.PermissionPlan {
		return false, false, false
	}
	if fromMode == types.PermissionPlan && toMode != types.PermissionPlan {
		return true, true, false
	}
	return false, false, false
}

// stripDangerousPermissionsForAutoMode moves dangerous session-source rules
// from alwaysAllowRules/alwaysDenyRules to strippedDangerousRules.
func stripDangerousPermissionsForAutoMode(ctx *types.ToolPermissionContextData) *types.ToolPermissionContextData {
	result := *ctx

	allowRules := cloneRulesMap(result.AlwaysAllowRules)
	denyRules := cloneRulesMap(result.AlwaysDenyRules)

	allowStripped := stripDangerousFromSource(allowRules, "session")
	denyStripped := stripDangerousFromSource(denyRules, "session")

	if allowStripped == nil && denyStripped == nil {
		return &result
	}

	stripped := map[string][]string{}
	if allowStripped != nil {
		stripped["alwaysAllowRules"] = mapSourceToSlice(allowStripped)
	}
	if denyStripped != nil {
		stripped["alwaysDenyRules"] = mapSourceToSlice(denyStripped)
	}

	raw, _ := json.Marshal(stripped)
	result.StrippedDangerousRules = raw

	result.AlwaysAllowRules = marshalRulesMap(allowRules)
	result.AlwaysDenyRules = marshalRulesMap(denyRules)

	return &result
}

// restoreDangerousPermissions restores rules from strippedDangerousRules back
// to alwaysAllowRules and alwaysDenyRules.
func restoreDangerousPermissions(ctx *types.ToolPermissionContextData) *types.ToolPermissionContextData {
	if len(ctx.StrippedDangerousRules) == 0 {
		return ctx
	}
	result := *ctx

	var stripped map[string][]string
	if err := json.Unmarshal(ctx.StrippedDangerousRules, &stripped); err != nil {
		return ctx
	}

	allowRules := cloneRulesMap(result.AlwaysAllowRules)
	denyRules := cloneRulesMap(result.AlwaysDenyRules)

	if srcRules, ok := stripped["alwaysAllowRules"]; ok {
		session := allowRules["session"]
		session = append(session, srcRules...)
		allowRules["session"] = session
	}
	if srcRules, ok := stripped["alwaysDenyRules"]; ok {
		session := denyRules["session"]
		session = append(session, srcRules...)
		denyRules["session"] = session
	}

	result.AlwaysAllowRules = marshalRulesMap(allowRules)
	result.AlwaysDenyRules = marshalRulesMap(denyRules)
	result.StrippedDangerousRules = nil

	return &result
}

// shouldPlanUseAutoMode checks if auto mode should be used during plan mode.
func shouldPlanUseAutoMode() bool {
	return growthbook.IsTenguPlanAutoMode()
}

// isAutoModeGateEnabled checks if the auto mode feature gate is currently on.
func isAutoModeGateEnabled() bool {
	return growthbook.IsTenguTranscriptClassifier()
}

// isDangerousRule checks if a single permission rule string matches a dangerous pattern.
func isDangerousRule(rule string) bool {
	trimmed := strings.TrimSpace(rule)
	for _, pattern := range dangerousRulePatterns {
		if strings.Contains(strings.ToLower(trimmed), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// stripDangerousFromSource removes dangerous rules from the given source in a rules map.
func stripDangerousFromSource(rules map[string][]string, source string) map[string][]string {
	srcRules, ok := rules[source]
	if !ok || len(srcRules) == 0 {
		return nil
	}
	var kept, removed []string
	for _, r := range srcRules {
		if isDangerousRule(r) {
			removed = append(removed, r)
		} else {
			kept = append(kept, r)
		}
	}
	if len(removed) == 0 {
		return nil
	}
	if len(kept) == 0 {
		delete(rules, source)
	} else {
		rules[source] = kept
	}
	return map[string][]string{source: removed}
}

// cloneRulesMap deep-copies a json.RawMessage-encoded permission rules map.
func cloneRulesMap(raw json.RawMessage) map[string][]string {
	if len(raw) == 0 || string(raw) == "{}" {
		return map[string][]string{}
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string][]string{}
	}
	return m
}

// marshalRulesMap marshals a rules map back to json.RawMessage.
func marshalRulesMap(m map[string][]string) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage(`{}`)
	}
	raw, _ := json.Marshal(m)
	return raw
}

// mapSourceToSlice converts a map[string][]string to a flat []string for storage.
func mapSourceToSlice(m map[string][]string) []string {
	var result []string
	for _, rules := range m {
		result = append(result, rules...)
	}
	return result
}
