// Package compactquerysource holds TS parity helpers for interpreting query source strings
// without importing the full query / hook graph (avoids compactservice → querycontext cycles).
package compactquerysource

import (
	"encoding/json"
	"strings"
)

// MainThreadLike mirrors TS runPostCompactCleanup isMainThreadCompact
// (undefined | repl_main_thread* | sdk). qs is often JSON-encoded from [json.RawMessage].
func MainThreadLike(qs string) bool {
	s := strings.TrimSpace(qs)
	if s == "" || s == "null" {
		return true
	}
	var decoded string
	if err := json.Unmarshal([]byte(s), &decoded); err == nil {
		s = strings.TrimSpace(decoded)
	} else if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	if s == "" || s == "sdk" {
		return true
	}
	return strings.HasPrefix(s, "repl_main_thread")
}
