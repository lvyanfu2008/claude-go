package handlers

import (
	"encoding/json"
)

// UsageResult is the JSON payload for /usage.
type UsageResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleUsageCommand handles /usage.
func HandleUsageCommand(args string) ([]byte, error) {
	return json.Marshal(UsageResult{
		Type: "text",
		Value: "API usage statistics are tracked per-session.\n" +
			"Use /cost for current session cost estimate.\n" +
			"Use /stats for session statistics.",
	})
}
