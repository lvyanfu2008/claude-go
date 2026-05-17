package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FastResult is the JSON payload for /fast.
type FastResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleFastCommand handles /fast [on|off].
func HandleFastCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	enabled := os.Getenv("CLAUDE_CODE_FAST_MODE") == "1"

	if args == "" {
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		return json.Marshal(FastResult{
			Type:  "text",
			Value: fmt.Sprintf("Fast mode is %s.\nUse /fast on or /fast off to toggle.", status),
		})
	}

	switch strings.ToLower(args) {
	case "on", "enable", "true", "1":
		os.Setenv("CLAUDE_CODE_FAST_MODE", "1")
		return json.Marshal(FastResult{Type: "text", Value: "Fast mode enabled."})
	case "off", "disable", "false", "0":
		os.Unsetenv("CLAUDE_CODE_FAST_MODE")
		return json.Marshal(FastResult{Type: "text", Value: "Fast mode disabled."})
	default:
		return json.Marshal(FastResult{Type: "text", Value: "Usage: /fast [on|off]"})
	}
}
