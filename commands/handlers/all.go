package handlers

import (
	"encoding/json"
)

// AllResult is the JSON payload returned by /all.
type AllResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleAllCommand returns the current allowed tools and permissions summary for /all.
// Mirrors TS /permissions (alias: allowed-tools).
func HandleAllCommand() ([]byte, error) {
	msg := AllResult{
		Type:  "text",
		Value: "gou-demo: /all is not fully supported. Use the TS CLI to review or modify tool permissions.\n\nFor gou-demo sessions, permission mode is controlled via GOU_PERMISSION_MODE env var:\n  - \"\": interactive (ask per tool call)\n  - \"allowed\": allow all\n  - \"denied\": deny all (read-only)",
	}
	return json.Marshal(msg)
}
