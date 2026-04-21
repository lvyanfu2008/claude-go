package hookexec

import (
	"encoding/json"
	"strings"
)

// collectCommandHooks returns command hooks from matcher groups.
// applyMatcherFilter mirrors whether TS had a defined matchQuery for filtering.
// When matchQuery is empty but applyMatcherFilter is true, TS treats matchQuery as falsy and skips filtering — we match that via doFilter.
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
