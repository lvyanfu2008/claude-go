package deferredtoolsdelta

import (
	"os"
	"strings"

	"goc/commands/featuregates"
	"goc/growthbook"
)

// Enabled mirrors isDeferredToolsDeltaEnabled (src/utils/toolSearch.ts): USER_TYPE=ant,
// GrowthBook tengu_glacier_2xr, plus Go-only CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA.
func Enabled() bool {
	if featuregates.UserTypeAnt() {
		return true
	}
	if growthbook.IsTenguGlacier2xr() {
		return true
	}
	return envTruthy("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA")
}

func envTruthy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
