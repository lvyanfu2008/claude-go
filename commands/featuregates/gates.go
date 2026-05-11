// Package featuregates mirrors TS bun:bundle feature() and env used by src/commands.ts / initBundledSkills.
package featuregates

import (
	"os"
	"sort"
	"strings"
)

func envTruthy(val string) bool {
	v := strings.TrimSpace(strings.ToLower(val))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// defaultTrueSet mirrors build.ts DEFAULT_BUILD_FEATURES — features that are
// always enabled at build time in TS and should be on by default in Go.
// Runtime env FEATURE_<NAME>=1 can enable additional features, but features
// in this set cannot be disabled at runtime (matching TS compile-time behaviour).
var defaultTrueSet = map[string]bool{
	// P0: local features
	"AGENT_TRIGGERS":              true,
	"ULTRATHINK":                  true,
	"BUILTIN_EXPLORE_PLAN_AGENTS": true,
	"LODESTONE":                   true,
	// P1: API-dependent features
	"EXTRACT_MEMORIES": true,
	"VERIFICATION_AGENT": true,
	"KAIROS_BRIEF":       true,
	"AWAY_SUMMARY":        true,
	"ULTRAPLAN":           true,
	// P2: daemon + remote control server
	"DAEMON": true,
	// PR-package restored features
	"WORKFLOW_SCRIPTS":         true,
	"HISTORY_SNIP":             true,
	"CONTEXT_COLLAPSE":         true,
	"MONITOR_TOOL":             true,
	"FORK_SUBAGENT":            true,
	"UDS_INBOX":                true,
	"KAIROS":                   true,
	"COORDINATOR_MODE":         true,
	"LAN_PIPES":                true,
	"POOR":                     true,
	"AGENT_TRIGGERS_REMOTE":    true,
	"CHICAGO_MCP":              true,
	"VOICE_MODE":               true,
	"SHOT_STATS":               true,
	"PROMPT_CACHE_BREAK_DETECTION": true,
	"TOKEN_BUDGET":                 true,
}

// Feature is true when the feature is in the TS-default set or when
// FEATURE_<name> is truthy (matching src/commands.ts: FEATURE_<FLAG_NAME>=1).
func Feature(name string) bool {
	if defaultTrueSet[name] {
		return true
	}
	return envTruthy(os.Getenv("FEATURE_" + name))
}

// UserTypeAnt matches process.env.USER_TYPE === 'ant'.
func UserTypeAnt() bool {
	return strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant"
}

// IsDemo matches truthy IS_DEMO.
func IsDemo() bool {
	return envTruthy(os.Getenv("IS_DEMO"))
}

// IsUsing3PServicesFromEnv is a Go shim for TS isUsing3PServices() when the host
// cannot evaluate real auth: set CLAUDE_CODE_GO_ASSUME_3P=1 to hide /login and /logout.
func IsUsing3PServicesFromEnv() bool {
	return envTruthy(os.Getenv("CLAUDE_CODE_GO_ASSUME_3P"))
}

// BundledChromeSkillEnabled replaces TS shouldAutoEnableClaudeInChrome() for listing metadata only.
func BundledChromeSkillEnabled() bool {
	return Feature("CHICAGO_MCP") || envTruthy(os.Getenv("CLAUDE_CODE_GO_BUNDLED_CHROME_SKILL"))
}

var fingerprintExtraKeys = []string{
	"CLAUDE_CODE_GO_ASSUME_3P",
	"CLAUDE_CODE_GO_BUNDLED_CHROME_SKILL",
	"IS_DEMO",
	"USER_TYPE",
}

// GatesFingerprint serializes env inputs that affect handwritten builtin/bundled assembly.
// Used for LoadAllCommands cache keys and invalidation of built-in name set caches in package commands.
func GatesFingerprint() string {
	env := os.Environ()
	pairs := make([]string, 0, len(env))
	for _, e := range env {
		name, _, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(name, "FEATURE_") {
			pairs = append(pairs, e)
			continue
		}
		for _, k := range fingerprintExtraKeys {
			if name == k {
				pairs = append(pairs, e)
				break
			}
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\x1e")
}
