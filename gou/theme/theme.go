// Package theme centralizes lipgloss colors for gou TUI (mirrors TS design-system role colors loosely).
package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"goc/types"
)

// MessageTypeColor maps transcript roles to ANSI256-ish lipgloss colors.
func MessageTypeColor(mt types.MessageType) lipgloss.Color {
	switch mt {
	case types.MessageTypeUser:
		return lipgloss.Color("39")
	case types.MessageTypeAssistant:
		return lipgloss.Color("141")
	default:
		return lipgloss.Color("245")
	}
}

// ToolError is used for tool_result bodies when is_error is true (TS OutputLine isError).
func ToolError() lipgloss.Color {
	return lipgloss.Color("196")
}

// ToolWarning matches TS warning tone for shell output.
func ToolWarning() lipgloss.Color {
	return lipgloss.Color("214")
}

// DimMuted is faint secondary text (headers, hints).
func DimMuted() lipgloss.Color {
	return lipgloss.Color("245")
}

// ParseThemeName maps settings / env theme string to a named preset (extend as TS themes are ported).
func ParseThemeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "default"
	}
	return s
}
