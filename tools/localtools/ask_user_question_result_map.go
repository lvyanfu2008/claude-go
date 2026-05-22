package localtools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MapAskUserQuestionOutputToToolResultContent mirrors AskUserQuestionTool.
// mapToolResultToToolResultBlockParam in TS. Formats the tool result as a readable
// summary string for the model.
func MapAskUserQuestionOutputToToolResultContent(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" {
		return "", fmt.Errorf("empty AskUserQuestion result")
	}
	var wrapper struct {
		Data struct {
			Answers     map[string]string `json:"answers"`
			Annotations map[string]struct {
				Preview string `json:"preview"`
				Notes   string `json:"notes"`
			} `json:"annotations"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &wrapper); err != nil {
		return "", fmt.Errorf("parsing AskUserQuestion result: %w", err)
	}
	answers := wrapper.Data.Answers
	if len(answers) == 0 {
		return "", fmt.Errorf("no answers in AskUserQuestion result")
	}

	sorted := make([]string, 0, len(answers))
	for q := range answers {
		sorted = append(sorted, q)
	}
	sort.Strings(sorted)

	var parts []string
	for _, questionText := range sorted {
		answer := answers[questionText]
		part := fmt.Sprintf("%q=%q", questionText, answer)
		if ann, ok := wrapper.Data.Annotations[questionText]; ok {
			if ann.Preview != "" {
				part += " selected preview:\n" + ann.Preview
			}
			if ann.Notes != "" {
				part += " user notes: " + ann.Notes
			}
		}
		parts = append(parts, part)
	}
	return fmt.Sprintf("User has answered your questions: %s. You can now continue with the user's answers in mind.",
		strings.Join(parts, ", ")), nil
}
