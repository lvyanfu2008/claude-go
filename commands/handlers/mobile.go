package handlers

import (
	"encoding/json"
)

// MobileResult is the JSON payload returned by /mobile.
type MobileResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleMobileCommand shows download links for the Claude mobile app for /mobile.
// Mirrors TS src/commands/mobile/ (local-jsx -> QR codes with platform switcher).
// gou-demo shows URLs without QR code rendering.
func HandleMobileCommand() ([]byte, error) {
	msg := MobileResult{
		Type: "text",
		Value: "Download the Claude mobile app:\n\n  iOS:     https://apps.apple.com/app/claude-by-anthropic/id6473753684\n  Android: https://play.google.com/store/apps/details?id=com.anthropic.claude",
	}
	return json.Marshal(msg)
}
