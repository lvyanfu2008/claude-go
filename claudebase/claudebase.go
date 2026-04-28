package claudebase

import "strings"

// Truthy mirrors the TS truthy helper: returns true for "1", "true", "yes", "on" (case-insensitive).
func Truthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// EnvDefinedFalsy returns true when s is non-empty and truthy-false ("0", "false", "no", "off").
func EnvDefinedFalsy(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "0" || v == "false" || v == "no" || v == "off"
}
