package querycontext

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// IsEnvTruthy mirrors src/utils/envUtils.ts isEnvTruthy.
func IsEnvTruthy(envVar string) bool {
	v := strings.TrimSpace(strings.ToLower(envVar))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// IsEnvDefinedFalsy mirrors src/utils/envUtils.ts isEnvDefinedFalsy.
func IsEnvDefinedFalsy(envVar string) bool {
	if envVar == "" {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(envVar))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// LocalISODate mirrors src/constants/common.ts getLocalISODate (local calendar date).
func LocalISODate() string {
	if o := strings.TrimSpace(os.Getenv("CLAUDE_CODE_OVERRIDE_DATE")); o != "" {
		return o
	}
	now := time.Now().Local()
	y, m, d := now.Date()
	return strconv.Itoa(y) + "-" + pad2(int(m)) + "-" + pad2(d)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// BareModeFromEnv matches isBareMode when only the env half is available (no argv in library code).
func BareModeFromEnv() bool {
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_SIMPLE"))
}

// FeatureBreakCacheCommand mirrors feature('BREAK_CACHE_COMMAND') via FEATURE_BREAK_CACHE_COMMAND=1.
func FeatureBreakCacheCommand() bool {
	return IsEnvTruthy(os.Getenv("FEATURE_BREAK_CACHE_COMMAND"))
}

// ShouldIncludeGitInstructions mirrors src/utils/gitSettings.ts for env-only Go hosts (settings.json omitted → default true).
func ShouldIncludeGitInstructions() bool {
	v, ok := os.LookupEnv("CLAUDE_CODE_DISABLE_GIT_INSTRUCTIONS")
	if !ok {
		return true
	}
	if IsEnvTruthy(v) {
		return false
	}
	if IsEnvDefinedFalsy(v) {
		return true
	}
	return true
}
