package hookexec

import (
	"encoding/json"
	"strings"
	"sync"

	"goc/tools/hookstypes"
)

// sessionStore holds session-scoped hooks keyed by session (or agent) ID.
// Mirrors TS SessionStore in utils/hooks/sessionHooks.ts.
type sessionStore struct {
	mu    sync.RWMutex
	hooks map[string]sessionHooksEntry // agentID → parsed hooks
}

type sessionHooksEntry struct {
	table HooksTable
}

var globalSessionStore = &sessionStore{
	hooks: make(map[string]sessionHooksEntry),
}

// SetSessionHooks stores parsed hooks for a session/agent ID.
func SetSessionHooks(agentID string, hooks HooksTable) {
	globalSessionStore.mu.Lock()
	defer globalSessionStore.mu.Unlock()
	globalSessionStore.hooks[agentID] = sessionHooksEntry{table: hooks}
}

// ClearSessionHooks removes hooks for a session/agent ID.
func ClearSessionHooks(agentID string) {
	globalSessionStore.mu.Lock()
	defer globalSessionStore.mu.Unlock()
	delete(globalSessionStore.hooks, agentID)
}

// GetSessionHooks returns the hooks table for a session/agent ID, or nil.
func GetSessionHooks(agentID string) HooksTable {
	globalSessionStore.mu.RLock()
	defer globalSessionStore.mu.RUnlock()
	entry, ok := globalSessionStore.hooks[agentID]
	if !ok {
		return nil
	}
	return entry.table
}

// MergeSessionHookTables merges all session hooks into a single HooksTable,
// keyed by event name. Used to blend session-scoped hooks with settings-file hooks.
// TS equivalent: getSessionHooks() in hooks.ts merges all session stores.
func MergeSessionHookTables(agentIDs ...string) HooksTable {
	globalSessionStore.mu.RLock()
	defer globalSessionStore.mu.RUnlock()

	var merged HooksTable
	for _, id := range agentIDs {
		entry, ok := globalSessionStore.hooks[id]
		if !ok {
			continue
		}
		merged = mergeHooksTable(merged, entry.table)
	}
	// Also merge all session hooks if no specific IDs given — TS behavior:
	// getSessionHooks iterates all entries in the sessionHooks Map.
	if len(agentIDs) == 0 {
		for _, entry := range globalSessionStore.hooks {
			merged = mergeHooksTable(merged, entry.table)
		}
	}
	return merged
}

// parseHooksRawToTable converts a json.RawMessage (agent frontmatter "hooks")
// into a HooksTable. Returns nil if the input is nil, empty, or unparseable.
func parseHooksRawToTable(raw json.RawMessage) HooksTable {
	if len(raw) == 0 {
		return nil
	}
	var rawTable map[string][]MatcherGroup
	if err := json.Unmarshal(raw, &rawTable); err != nil {
		return nil
	}
	if len(rawTable) == 0 {
		return nil
	}
	table := make(HooksTable, len(rawTable))
	for event, matchers := range rawTable {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		// Validate event name via KnownHookEvent (mirrors TS HooksSchema.safeParse).
		if !hookstypes.KnownHookEvent(event) {
			continue
		}
		cleaned := make([]MatcherGroup, 0, len(matchers))
		for _, mg := range matchers {
			if len(mg.Hooks) == 0 {
				continue
			}
			cleaned = append(cleaned, mg)
		}
		if len(cleaned) > 0 {
			table[event] = cleaned
		}
	}
	if len(table) == 0 {
		return nil
	}
	return table
}
