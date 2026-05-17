package permissionrules

import (
	"encoding/json"

	"goc/types"
)

// PermissionUpdate mirrors TS PermissionUpdate type in src/types/permissions.ts.
// It describes a mutation to the permission context: adding/removing/replacing rules
// or changing the permission mode.
type PermissionUpdate struct {
	Type        string `json:"type"`        // "addRules" | "replaceRules" | "removeRules" | "setMode"
	Destination string `json:"destination"` // "userSettings" | "projectSettings" | "localSettings" | "session" | ...
	Behavior    string `json:"behavior"`    // "allow" | "deny" | "ask" (for rule updates)
	Rules       []string `json:"rules,omitempty"`
	Mode        string   `json:"mode,omitempty"` // for setMode
}

// ApplyPermissionUpdate applies a single PermissionUpdate to a ToolPermissionContextData in place.
// Mirrors TS applyPermissionUpdate in src/utils/permissions/PermissionUpdate.ts.
func ApplyPermissionUpdate(ctx *types.ToolPermissionContextData, update PermissionUpdate) {
	switch update.Type {
	case "setMode":
		ctx.Mode = types.PermissionMode(update.Mode)
	case "addRules":
		applyAddRules(ctx, update)
	case "replaceRules":
		applyReplaceRules(ctx, update)
	case "removeRules":
		applyRemoveRules(ctx, update)
	}
}

// ApplyPermissionUpdates applies a batch of PermissionUpdates sequentially.
func ApplyPermissionUpdates(ctx *types.ToolPermissionContextData, updates []PermissionUpdate) {
	for _, u := range updates {
		ApplyPermissionUpdate(ctx, u)
	}
}

func applyAddRules(ctx *types.ToolPermissionContextData, update PermissionUpdate) {
	dest := update.Destination
	rules := update.Rules
	if len(rules) == 0 {
		return
	}
	switch update.Behavior {
	case "allow":
		ctx.AlwaysAllowRules = mergeRules(ctx.AlwaysAllowRules, dest, rules)
	case "deny":
		ctx.AlwaysDenyRules = mergeRules(ctx.AlwaysDenyRules, dest, rules)
	case "ask":
		ctx.AlwaysAskRules = mergeRules(ctx.AlwaysAskRules, dest, rules)
	}
}

func applyReplaceRules(ctx *types.ToolPermissionContextData, update PermissionUpdate) {
	dest := update.Destination
	rules := update.Rules
	switch update.Behavior {
	case "allow":
		ctx.AlwaysAllowRules = replaceRules(ctx.AlwaysAllowRules, dest, rules)
	case "deny":
		ctx.AlwaysDenyRules = replaceRules(ctx.AlwaysDenyRules, dest, rules)
	case "ask":
		ctx.AlwaysAskRules = replaceRules(ctx.AlwaysAskRules, dest, rules)
	}
}

func applyRemoveRules(ctx *types.ToolPermissionContextData, update PermissionUpdate) {
	dest := update.Destination
	rules := update.Rules
	if len(rules) == 0 {
		return
	}
	removeSet := make(map[string]struct{}, len(rules))
	for _, r := range rules {
		removeSet[r] = struct{}{}
	}
	switch update.Behavior {
	case "allow":
		ctx.AlwaysAllowRules = removeRulesFromSource(ctx.AlwaysAllowRules, dest, removeSet)
	case "deny":
		ctx.AlwaysDenyRules = removeRulesFromSource(ctx.AlwaysDenyRules, dest, removeSet)
	case "ask":
		ctx.AlwaysAskRules = removeRulesFromSource(ctx.AlwaysAskRules, dest, removeSet)
	}
}

// mergeRules appends newRules to the source's rule list. If the source or existing
// rules don't exist, a new entry is created.
func mergeRules(raw json.RawMessage, source string, newRules []string) json.RawMessage {
	m := parseRulesMap(raw)
	existing := m[source]
	seen := make(map[string]struct{}, len(existing)+len(newRules))
	for _, r := range existing {
		seen[r] = struct{}{}
	}
	for _, r := range newRules {
		if _, ok := seen[r]; !ok {
			existing = append(existing, r)
			seen[r] = struct{}{}
		}
	}
	m[source] = existing
	b, _ := json.Marshal(m)
	return json.RawMessage(b)
}

// replaceRules replaces the entire rule list for a source.
func replaceRules(raw json.RawMessage, source string, rules []string) json.RawMessage {
	m := parseRulesMap(raw)
	m[source] = rules
	b, _ := json.Marshal(m)
	return json.RawMessage(b)
}

// removeRulesFromSource removes specific rules from a source's rule list.
func removeRulesFromSource(raw json.RawMessage, source string, removeSet map[string]struct{}) json.RawMessage {
	m := parseRulesMap(raw)
	existing := m[source]
	if len(existing) == 0 {
		return raw
	}
	out := make([]string, 0, len(existing))
	for _, r := range existing {
		if _, ok := removeSet[r]; !ok {
			out = append(out, r)
		}
	}
	m[source] = out
	b, _ := json.Marshal(m)
	return json.RawMessage(b)
}

func parseRulesMap(raw json.RawMessage) map[string][]string {
	if len(raw) == 0 || string(raw) == "null" {
		return make(map[string][]string)
	}
	var m map[string][]string
	if json.Unmarshal(raw, &m) != nil {
		return make(map[string][]string)
	}
	if m == nil {
		return make(map[string][]string)
	}
	return m
}
