package hookexec

import (
	"encoding/json"
	"strings"

	"goc/tools/hookstypes"
)

// collectCommandHooks returns command hooks from matcher groups.
// applyMatcherFilter mirrors whether TS had a defined matchQuery for filtering.
func collectCommandHooks(matchers []MatcherGroup, matchQuery string, applyMatcherFilter bool) []commandHook {
	doFilter := applyMatcherFilter && strings.TrimSpace(matchQuery) != ""
	var out []commandHook
	for _, mg := range matchers {
		if doFilter && !MatchesPattern(matchQuery, mg.Matcher) {
			continue
		}
		for _, raw := range mg.Hooks {
			var h commandHook
			if err := json.Unmarshal(raw, &h); err != nil {
				continue
			}
			if strings.TrimSpace(h.Type) != "command" || strings.TrimSpace(h.Command) == "" {
				continue
			}
			out = append(out, h)
		}
	}
	return out
}

// collectAllHooks returns all hooks (command, prompt, http, agent) from matcher groups.
// TS parity: getMatchingHooks in hooks.ts — does not filter by type.
func collectAllHooks(matchers []MatcherGroup, matchQuery string, applyMatcherFilter bool) []hookstypes.HookCommand {
	doFilter := applyMatcherFilter && strings.TrimSpace(matchQuery) != ""
	var out []hookstypes.HookCommand
	for _, mg := range matchers {
		if doFilter && !MatchesPattern(matchQuery, mg.Matcher) {
			continue
		}
		for _, raw := range mg.Hooks {
			var h hookstypes.HookCommand
			if err := json.Unmarshal(raw, &h); err != nil {
				continue
			}
			t := strings.TrimSpace(h.Type)
			if t == "" {
				continue
			}
			// Validate the hook has its required payload field.
			switch t {
			case "command":
				if strings.TrimSpace(h.Command) == "" {
					continue
				}
			case "prompt":
				if strings.TrimSpace(h.Prompt) == "" {
					continue
				}
			case "http":
				if strings.TrimSpace(h.URL) == "" {
					continue
				}
			case "agent":
				if strings.TrimSpace(h.Prompt) == "" {
					continue
				}
			default:
				continue
			}
			out = append(out, h)
		}
	}
	return out
}

// CommandHooksForHookInput selects command hooks using the same matchQuery rules as TS getMatchingHooks.
func CommandHooksForHookInput(table HooksTable, hookInput map[string]any) []commandHook {
	ev, _ := hookInput["hook_event_name"].(string)
	ev = strings.TrimSpace(ev)
	if ev == "" {
		return nil
	}
	mq, use := DeriveMatchQuery(hookInput)
	return collectCommandHooks(table[ev], mq, use)
}

// AllHooksForHookInput selects all hooks (any type) using matchQuery rules.
// TS parity: getMatchingHooks in hooks.ts (without the type filter).
func AllHooksForHookInput(table HooksTable, hookInput map[string]any) []hookstypes.HookCommand {
	ev, _ := hookInput["hook_event_name"].(string)
	ev = strings.TrimSpace(ev)
	if ev == "" {
		return nil
	}
	mq, use := DeriveMatchQuery(hookInput)
	return collectAllHooks(table[ev], mq, use)
}
