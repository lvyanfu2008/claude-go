package config

import (
	"os"
	"strings"
)

// EnvTruthy returns true when the env var value is truthy.
func EnvTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// EnvFalsy returns true when the env var value is falsy.
func EnvFalsy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// StatusLineEnabled returns true when GOU_DEMO_STATUS_LINE is set.
func StatusLineEnabled() bool {
	return EnvTruthy("GOU_DEMO_STATUS_LINE")
}

// EnvWantsAPIBodyLog returns true when CLAUDE_CODE_LOG_API_REQUEST_BODY or CLAUDE_CODE_LOG_API_RESPONSE_BODY is set.
func EnvWantsAPIBodyLog() bool {
	return EnvTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") || EnvTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY")
}
