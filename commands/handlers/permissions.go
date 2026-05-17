package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PermissionsResult is the JSON payload for /permissions.
type PermissionsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandlePermissionsCommand handles /permissions.
func HandlePermissionsCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	mode := os.Getenv("CLAUDE_CODE_PERMISSION_MODE")
	if mode == "" {
		mode = "default"
	}

	if args == "" {
		return json.Marshal(PermissionsResult{
			Type:  "text",
			Value: fmt.Sprintf("Current permission mode: %s\nUse /permissions [default|acceptEdits|bypassPermissions|plan|auto] to change.", mode),
		})
	}

	switch strings.ToLower(args) {
	case "default", "acceptEdits", "bypassPermissions", "plan", "auto":
		os.Setenv("CLAUDE_CODE_PERMISSION_MODE", strings.ToLower(args))
		return json.Marshal(PermissionsResult{
			Type:  "text",
			Value: fmt.Sprintf("Permission mode set to: %s", strings.ToLower(args)),
		})
	default:
		return json.Marshal(PermissionsResult{
			Type:  "text",
			Value: fmt.Sprintf("Unknown permission mode: %s\nValid modes: default, acceptEdits, bypassPermissions, plan, auto", args),
		})
	}
}
