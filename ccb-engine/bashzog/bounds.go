package bashzog

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultBashTimeoutMs = 120_000
	maxBashTimeoutMsCap  = 600_000
)

func defaultBashTimeoutMsFromEnv() int {
	v := strings.TrimSpace(os.Getenv("BASH_DEFAULT_TIMEOUT_MS"))
	if v == "" {
		return defaultBashTimeoutMs
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultBashTimeoutMs
	}
	return n
}

// maxBashTimeoutMs mirrors claude-code src/utils/timeouts.ts getMaxBashTimeoutMs.
func maxBashTimeoutMs() int {
	def := defaultBashTimeoutMsFromEnv()
	v := strings.TrimSpace(os.Getenv("BASH_MAX_TIMEOUT_MS"))
	if v == "" {
		return maxInt(maxBashTimeoutMsCap, def)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return maxInt(maxBashTimeoutMsCap, def)
	}
	return maxInt(n, def)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
