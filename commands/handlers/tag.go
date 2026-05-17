package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TagResult is the JSON payload for /tag.
type TagResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleTagCommand handles /tag [tag-name].
func HandleTagCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	current := os.Getenv("CLAUDE_SESSION_TAG")

	if args == "" {
		if current == "" {
			return json.Marshal(TagResult{
				Type:  "text",
				Value: "No tag set for current session.\nUse /tag [tag-name] to add a tag.",
			})
		}
		return json.Marshal(TagResult{
			Type:  "text",
			Value: fmt.Sprintf("Current session tag: %s", current),
		})
	}

	if args == "clear" || args == "remove" || args == "none" {
		os.Unsetenv("CLAUDE_SESSION_TAG")
		return json.Marshal(TagResult{Type: "text", Value: "Tag cleared."})
	}

	os.Setenv("CLAUDE_SESSION_TAG", args)
	return json.Marshal(TagResult{
		Type:  "text",
		Value: fmt.Sprintf("Session tagged: %s", args),
	})
}
