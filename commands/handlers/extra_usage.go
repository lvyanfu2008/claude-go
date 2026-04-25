package handlers

import (
	"encoding/json"
)

// ExtraUsageResult is the JSON payload returned by /extra-usage.
type ExtraUsageResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleExtraUsageCommand returns usage limit info for /extra-usage.
// Mirrors TS src/commands/extra-usage/ (local-jsx with interactive upsell UI).
// gou-demo provides static info only; the web dashboard handles activation.
func HandleExtraUsageCommand() ([]byte, error) {
	msg := ExtraUsageResult{
		Type:  "text",
		Value: "Extra usage allows you to keep working when you hit rate or usage limits.\n\n" +
			"To configure extra usage, visit:\n" +
			"  https://claude.ai/settings/billing\n\n" +
			"In the TS CLI, /extra-usage opens an interactive upsell flow.\n" +
			"gou-demo does not support configuring extra usage directly.",
	}
	return json.Marshal(msg)
}
