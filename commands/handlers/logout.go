package handlers

import (
	"encoding/json"
	"os"
)

// LogoutResult is the JSON payload for /logout.
type LogoutResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleLogoutCommand handles /logout.
func HandleLogoutCommand(args string) ([]byte, error) {
	os.Unsetenv("CLAUDE_CODE_OAUTH_ACCESS_TOKEN")
	os.Unsetenv("CLAUDE_CODE_OAUTH_REFRESH_TOKEN")

	return json.Marshal(LogoutResult{
		Type: "text",
		Value: "Logged out. OAuth tokens have been cleared from this session.\n" +
			"Use /login to authenticate again.\n" +
			"Note: persistent tokens in keychain may still be available on restart.",
	})
}
