package handlers

import (
	"encoding/json"
	"os"
)

// SessionResult is the JSON payload returned by /session.
type SessionResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleSessionCommand shows the remote session URL for /session.
// Mirrors TS src/commands/session/ (local-jsx -> SessionInfo with QR code).
// gou-demo shows the URL without QR code rendering.
func HandleSessionCommand() ([]byte, error) {
	remoteURL := os.Getenv("CLAUDE_CODE_REMOTE_SESSION_URL")
	sessionID := os.Getenv("CLAUDE_SESSION_ID")

	if remoteURL == "" {
		return json.Marshal(SessionResult{
			Type:  "text",
			Value: "Not in remote mode. Start with `claude --remote` to use this command.\n\nSession ID: " + sessionID,
		})
	}

	return json.Marshal(SessionResult{
		Type:  "text",
		Value: "Remote session:\n  URL: " + remoteURL + "\n\nOpen in browser to connect remotely.",
	})
}
