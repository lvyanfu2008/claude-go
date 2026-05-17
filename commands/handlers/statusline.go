package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// StatuslineResult is the JSON payload for /statusline.
type StatuslineResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleStatuslineCommand handles /statusline [on|off].
func HandleStatuslineCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	enabled := os.Getenv("CLAUDE_CODE_STATUSLINE") != "0"

	if args == "" {
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		return json.Marshal(StatuslineResult{
			Type:  "text",
			Value: fmt.Sprintf("Status line is %s.\nUse /statusline on or /statusline off to toggle.", status),
		})
	}

	switch strings.ToLower(args) {
	case "on", "enable", "true", "1":
		os.Unsetenv("CLAUDE_CODE_STATUSLINE")
		return json.Marshal(StatuslineResult{Type: "text", Value: "Status line enabled."})
	case "off", "disable", "false", "0":
		os.Setenv("CLAUDE_CODE_STATUSLINE", "0")
		return json.Marshal(StatuslineResult{Type: "text", Value: "Status line disabled."})
	default:
		return json.Marshal(StatuslineResult{Type: "text", Value: "Usage: /statusline [on|off]"})
	}
}
