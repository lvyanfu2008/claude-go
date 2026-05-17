package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PRCommentsResult is the JSON payload for /pr-comments.
type PRCommentsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandlePRCommentsCommand handles /pr-comments.
func HandlePRCommentsCommand(args string) ([]byte, error) {
	prNumber := os.Getenv("CLAUDE_CODE_PR_NUMBER")
	prURL := os.Getenv("CLAUDE_CODE_PR_URL")

	if prNumber == "" && prURL == "" {
		return json.Marshal(PRCommentsResult{
			Type: "text",
			Value: "No active PR linked to this session.\n" +
				"Use --from-pr [PR number/URL] to start a session linked to a PR.",
		})
	}

	var lines []string
	lines = append(lines, "PR information:")
	if prNumber != "" {
		lines = append(lines, fmt.Sprintf("  PR Number: %s", prNumber))
	}
	if prURL != "" {
		lines = append(lines, fmt.Sprintf("  PR URL: %s", prURL))
	}
	lines = append(lines, "\nUse `gh pr view --comments` to see PR comments.")
	return json.Marshal(PRCommentsResult{Type: "text", Value: strings.Join(lines, "\n")})
}
