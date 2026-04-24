// Package modelenv resolves LLM model id from process environment (Claude Code / gou-demo / ccb-engine).
// Kept at module root (not under goc/internal/anthropic) so goc/commands and goc/querycontext can import it without the Messages client stack.
package modelenv

import (
	"os"
	"strings"
)

// DefaultMainLoopModelID is used when no env key in [LookupKeys] is set (matches [gou/pui] demo default).
const DefaultMainLoopModelID = "claude-sonnet-4-20250514"

// LookupKeys is the env precedence for HTTP model id and (before Gou.ModelID) system # Environment.
// CLAUDE_CODE_MODEL first so /model and merged Claude Code settings override generic ANTHROPIC_* defaults.
var LookupKeys = []string{
	"CLAUDE_CODE_MODEL",
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

// EffectiveMainLoopModel returns the model id for the next API turn: env chain ([LookupKeys]) or [DefaultMainLoopModelID].
func EffectiveMainLoopModel() string {
	return ResolveWithFallback(DefaultMainLoopModelID)
}
