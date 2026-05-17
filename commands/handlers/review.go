package handlers

import (
	"encoding/json"
)

// ReviewResult is the JSON payload for /review.
type ReviewResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleReviewCommand handles /review — triggers code review.
func HandleReviewCommand(args string) ([]byte, error) {
	return json.Marshal(ReviewResult{
		Type: "text",
		Value: "Code review requested.\n" +
			"Claude will review recent changes in the next turn.\n" +
			"Use a prompt like 'review the changes in this file' for targeted review.",
	})
}
