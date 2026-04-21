package hookexec

import (
	"os"
	"strings"
)

func envTruthy(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// HooksDisabled mirrors TS executeHooks early-return under bare/simple mode (no querycontext import — avoids cycles).
func HooksDisabled() bool {
	if envTruthy(os.Getenv("CLAUDE_CODE_SIMPLE")) {
		return true
	}
	return envTruthy(os.Getenv("CLAUDE_CODE_BARE"))
}
