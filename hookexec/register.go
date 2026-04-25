package hookexec

import (
	"encoding/json"
	"strings"

	"goc/tools/hookstypes"
)

// RegisterFrontmatterHooks parses the raw hooks value from agent (or skill)
// frontmatter and registers them as session-scoped hooks for the given agent ID.
//
// Mirrors TS registerFrontmatterHooks() in registerFrontmatterHooks.ts:
//   - Iterates all HOOK_EVENTS, matches against the hooks config
//   - When isAgent is true, any "Stop" hooks are remapped to "SubagentStop"
//     (subagents trigger SubagentStop, not Stop)
//   - Stores via addSessionHook (SetSessionHooks in Go)
//
// The hooksTable is stored in the global session store keyed by agentID.
func RegisterFrontmatterHooks(agentID string, hooksRaw json.RawMessage, isAgent bool) {
	table := parseHooksRawToTable(hooksRaw)
	if table == nil {
		return
	}

	// Apply Stop → SubagentStop conversion for agent hooks.
	// TS: if (isAgent && event === 'Stop') { event = 'SubagentStop'; }
	if isAgent {
		if stops, ok := table[hookEventStop]; ok {
			table[hookEventSubagentStop] = append(table[hookEventSubagentStop], stops...)
			delete(table, hookEventStop)
		}
	}

	// Additionally, apply Stop → SubagentStop for any event that matches the
	// string "Stop" within AllHookEvents — TS iterates HOOK_EVENTS and remaps.
	// We do this at the table level rather than individually per hook.
	normalized := make(HooksTable)
	for event, matchers := range table {
		e := strings.TrimSpace(event)
		if isAgent && strings.EqualFold(e, string(hookEventStop)) {
			e = string(hookEventSubagentStop)
		}
		normalized[e] = matchers
	}

	SetSessionHooks(agentID, normalized)
}

const (
	hookEventStop          = string(hookstypes.Stop)
	hookEventSubagentStop  = string(hookstypes.SubagentStop)
)

// ClearAgentSessionHooks is a convenience wrapper that removes session hooks
// for a given agent ID. Should be called in defer after agent execution.
func ClearAgentSessionHooks(agentID string) {
	ClearSessionHooks(agentID)
}
