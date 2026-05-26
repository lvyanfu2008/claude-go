package handlers

import (
	"encoding/json"
)

// LoginResult is the JSON payload for /login.
type LoginResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleLoginCommand handles /login.
func HandleLoginCommand(args string) ([]byte, error) {
	return json.Marshal(LoginResult{
		Type: "text",
		Value: "To authenticate with Harness Code:\n" +
			"  1. Run: harness auth login\n" +
			"  2. Follow the browser prompts\n" +
			"  3. Return to this session\n\n" +
			"Or set CLAUDE_CODE_OAUTH_ACCESS_TOKEN to use an existing token.",
	})
}
