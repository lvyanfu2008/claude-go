package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ThemeResult is the JSON payload for /theme.
type ThemeResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleThemeCommand handles /theme [theme-name|default].
func HandleThemeCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	current := os.Getenv("CLAUDE_CODE_THEME")
	if current == "" {
		current = "default"
	}

	if args == "" {
		return json.Marshal(ThemeResult{
			Type:  "text",
			Value: fmt.Sprintf("Current theme: %s\nUse /theme [theme-name] to change, or /theme default to reset.", current),
		})
	}

	if args == "default" {
		os.Unsetenv("CLAUDE_CODE_THEME")
		return json.Marshal(ThemeResult{
			Type:  "text",
			Value: "Theme reset to default.",
		})
	}

	os.Setenv("CLAUDE_CODE_THEME", args)
	return json.Marshal(ThemeResult{
		Type:  "text",
		Value: fmt.Sprintf("Theme set to: %s", args),
	})
}
