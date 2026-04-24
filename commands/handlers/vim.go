package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// VimResult is the JSON payload returned by /vim.
type VimResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleVimCommand toggles or checks the editor mode (vim keybindings).
// Mirrors TS src/commands/vim.ts — checks ccb-editor-mode in settings.
// In Go, we check CLAUDE_CODE_EDITOR_MODE env var as a proxy.
func HandleVimCommand(args string) ([]byte, error) {
	current := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_EDITOR_MODE")))
	enabled := current == "1" || current == "true" || current == "vim" || current == "yes" || current == "on"

	if args != "" {
		v := strings.TrimSpace(strings.ToLower(args))
		switch v {
		case "on", "enable", "1", "true", "yes":
			if !enabled {
				_ = os.Setenv("CLAUDE_CODE_EDITOR_MODE", "1")
			}
			return json.Marshal(VimResult{
				Type:  "text",
				Value: "Vim mode enabled for this session.",
			})
		case "off", "disable", "0", "false", "no":
			if enabled {
				_ = os.Unsetenv("CLAUDE_CODE_EDITOR_MODE")
			}
			return json.Marshal(VimResult{
				Type:  "text",
				Value: "Vim mode disabled for this session.",
			})
		default:
			return json.Marshal(VimResult{
				Type:  "text",
				Value: fmt.Sprintf("Usage: /vim [on|off]. Current: %v", enabled),
			})
		}
	}

	if enabled {
		return json.Marshal(VimResult{Type: "text", Value: "Vim mode is currently enabled."})
	}
	return json.Marshal(VimResult{Type: "text", Value: "Vim mode is currently disabled."})
}
