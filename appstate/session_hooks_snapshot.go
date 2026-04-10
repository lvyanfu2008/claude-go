package appstate

import "encoding/json"

// SessionHookEntrySnapshot mirrors SessionHookMatcher.hooks[] elements ({ hook, onHookSuccess? }).
// onHookSuccess is a callback and is never present in JSON.
type SessionHookEntrySnapshot struct {
	Hook json.RawMessage `json:"hook"`
}

// SessionHookMatcherSnapshot mirrors src/utils/hooks/sessionHooks.ts SessionHookMatcher for
// command hooks only (FunctionHook callbacks are never JSON-serializable).
type SessionHookMatcherSnapshot struct {
	Matcher   string                     `json:"matcher"`
	SkillRoot string                     `json:"skillRoot,omitempty"`
	Hooks     []SessionHookEntrySnapshot `json:"hooks"`
}

// SessionStoreSnapshot mirrors SessionStore (hooks keyed by HookEvent string).
type SessionStoreSnapshot struct {
	Hooks map[string][]SessionHookMatcherSnapshot `json:"hooks"`
}

// SessionHooksState mirrors AppState sessionHooks: Map<sessionId, SessionStore>.
type SessionHooksState map[string]SessionStoreSnapshot

// SanitizeSessionStoreSnapshot fills nil maps/slices so JSON matches TS empty structures.
func SanitizeSessionStoreSnapshot(v SessionStoreSnapshot) SessionStoreSnapshot {
	h := v.Hooks
	if h == nil {
		h = make(map[string][]SessionHookMatcherSnapshot)
	}
	for ev, matchers := range h {
		if matchers == nil {
			h[ev] = []SessionHookMatcherSnapshot{}
			continue
		}
		for i := range matchers {
			if matchers[i].Hooks == nil {
				matchers[i].Hooks = []SessionHookEntrySnapshot{}
			}
		}
		h[ev] = matchers
	}
	return SessionStoreSnapshot{Hooks: h}
}

// MarshalJSON encodes nil map as {} and normalizes inner snapshots.
func (s SessionHooksState) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("{}"), nil
	}
	out := make(map[string]SessionStoreSnapshot, len(s))
	for k, v := range s {
		out[k] = SanitizeSessionStoreSnapshot(v)
	}
	return json.Marshal(out)
}

// UnmarshalJSON normalizes inner structures after decode.
func (s *SessionHooksState) UnmarshalJSON(b []byte) error {
	var raw map[string]SessionStoreSnapshot
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw == nil {
		*s = SessionHooksState{}
		return nil
	}
	sh := SessionHooksState(raw)
	for k, v := range sh {
		sh[k] = SanitizeSessionStoreSnapshot(v)
	}
	*s = sh
	return nil
}
