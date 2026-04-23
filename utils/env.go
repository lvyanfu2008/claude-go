package utils

import (
	"os"
	"strings"
)

// IsEnvTruthy matches TS isEnvTruthy: present + not "0"/"false"/empty-after-trim.
func IsEnvTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}