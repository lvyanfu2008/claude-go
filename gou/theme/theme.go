// Package theme centralizes lipgloss colors for gou TUI (mirrors TS design-system roles loosely).
package theme

import "strings"

// ParseThemeName maps settings / env theme string to a named preset (extend as TS themes are ported).
func ParseThemeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "default"
	}
	return s
}
