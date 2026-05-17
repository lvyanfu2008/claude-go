package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AddDirResult is the JSON payload for /add-dir.
type AddDirResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleAddDirCommand handles /add-dir [path].
func HandleAddDirCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	current := os.Getenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES")

	if args == "" {
		if current == "" {
			return json.Marshal(AddDirResult{
				Type:  "text",
				Value: "No additional directories configured.\nUse /add-dir [path] to add a directory for tool access.",
			})
		}
		return json.Marshal(AddDirResult{
			Type:  "text",
			Value: fmt.Sprintf("Additional directories:\n%s", strings.ReplaceAll(current, ":", "\n")),
		})
	}

	if current == "" {
		os.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES", args)
	} else {
		os.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES", current+":"+args)
	}
	return json.Marshal(AddDirResult{
		Type:  "text",
		Value: fmt.Sprintf("Added directory: %s", args),
	})
}
