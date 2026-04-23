package toolpool

import (
	"os"
	"strings"
)

// FeatureCoordinatorMode mirrors feature('COORDINATOR_MODE') (see scripts/defines.ts; enable via FEATURE_COORDINATOR_MODE=1).
func FeatureCoordinatorMode() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_COORDINATOR_MODE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// IsCoordinatorMode mirrors coordinatorModeModule.isCoordinatorMode() — TS AgentTool uses CLAUDE_CODE_COORDINATOR_MODE (src/tools/AgentTool/AgentTool.tsx).
func IsCoordinatorMode() bool {
	return isEnvTruthy(os.Getenv("CLAUDE_CODE_COORDINATOR_MODE"))
}

// CoordinatorMergeFilterActive mirrors feature('COORDINATOR_MODE') && coordinatorModeModule.isCoordinatorMode() in mergeAndFilterTools (src/utils/toolPool.ts lines 72–75).
func CoordinatorMergeFilterActive() bool {
	return FeatureCoordinatorMode() && IsCoordinatorMode()
}

func isEnvTruthy(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
