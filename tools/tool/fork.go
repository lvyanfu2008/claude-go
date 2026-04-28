package tool

import (
	"goc/commands/featuregates"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// IsForkSubagentEnabled mirrors forkSubagent.ts isForkSubagentEnabled.
// Gate: FEATURE_FORK_SUBAGENT=1, excludes coordinator mode and non-interactive sessions.
// Matches commands.ForkSubagentEnabled — standalone variant for tools package callers.
func IsForkSubagentEnabled() bool {
	if coordinatorModeEnvShim() {
		return false
	}
	if nonInteractiveSessionEnvShim() {
		return false
	}
	return true
}

// coordinatorModeEnvShim mirrors coordinatorModeLikeTS in commands/prompts_gates.go.
// Checks FEATURE_COORDINATOR_MODE + CLAUDE_CODE_COORDINATOR_MODE.
func coordinatorModeEnvShim() bool {
	if !featuregates.Feature("COORDINATOR_MODE") {
		return false
	}
	return EnvTruthy("CLAUDE_CODE_COORDINATOR_MODE")
}

// nonInteractiveSessionEnvShim approximates getIsNonInteractiveSession() from TS
// bootstrap/state.js when no session struct is in scope.
// TS checks: -p/--print, --init-only, --sdk-url, or !process.stdout.isTTY
// Go mirrors via env vars + stdout terminal detection.
func nonInteractiveSessionEnvShim() bool {
	return EnvTruthy("CLAUDE_CODE_NONINTERACTIVE") ||
		EnvTruthy("HEADLESS") ||
		EnvTruthy("GOU_DEMO_NON_INTERACTIVE") ||
		!isatty.IsTerminal(os.Stdout.Fd())
}

func EnvTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
