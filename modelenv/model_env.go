// Package modelenv resolves LLM model id from process environment (Claude Code / gou-demo / ccb-engine).
// Kept outside ccb-engine/internal so goc/commands and goc/querycontext can import it.
package modelenv

import (
	"os"
	"strings"
)

// LookupKeys is the env precedence for HTTP model id and (before Gou.ModelID) system # Environment.
var LookupKeys = []string{
	"CCB_ENGINE_MODEL",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
}

// FirstNonEmpty returns the first non-empty trimmed value among [LookupKeys], or "".
func FirstNonEmpty() string {
	for _, k := range LookupKeys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// ResolveWithFallback returns [FirstNonEmpty] or trimmed fallback when all keys are empty.
func ResolveWithFallback(fallbackWhenUnset string) string {
	if v := FirstNonEmpty(); v != "" {
		return v
	}
	return strings.TrimSpace(fallbackWhenUnset)
}
