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

// Feature is true when FEATURE_<name> is truthy (AGENTS.md: FEATURE_<FLAG_NAME>=1).
func Feature(name string) bool {
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
// Used for LoadAllCommands cache keys and BuiltinCommandNameSet invalidation.
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
