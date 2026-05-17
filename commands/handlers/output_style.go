package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// OutputStyleResult is the JSON payload for /output-style.
type OutputStyleResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleOutputStyleCommand handles /output-style [text|json|stream-json].
func HandleOutputStyleCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	current := os.Getenv("CLAUDE_CODE_OUTPUT_FORMAT")
	if current == "" {
		current = "text"
	}

	if args == "" {
		return json.Marshal(OutputStyleResult{
			Type:  "text",
			Value: fmt.Sprintf("Current output style: %s\nUse /output-style [text|json|stream-json] to change.", current),
		})
	}

	valid := map[string]bool{"text": true, "json": true, "stream-json": true}
	if !valid[strings.ToLower(args)] {
		return json.Marshal(OutputStyleResult{
			Type:  "text",
			Value: "Invalid output style. Valid: text, json, stream-json",
		})
	}

	os.Setenv("CLAUDE_CODE_OUTPUT_FORMAT", strings.ToLower(args))
	return json.Marshal(OutputStyleResult{
		Type:  "text",
		Value: fmt.Sprintf("Output style set to: %s", strings.ToLower(args)),
	})
}
