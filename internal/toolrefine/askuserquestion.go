package toolrefine

import (
	"encoding/json"
	"fmt"
)

// ValidateAskUserQuestionUniqueness mirrors AskUserQuestionTool.tsx UNIQUENESS_REFINE
// (Zod .refine) — JSON Schema from toolToAPISchema does not encode duplicate checks.
func ValidateAskUserQuestionUniqueness(input json.RawMessage) error {
	var p struct {
		Questions []struct {
			Question string `json:"question"`
			Options  []struct {
				Label string `json:"label"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(input, &p); err != nil {
		return nil
	}
	qs := make([]string, 0, len(p.Questions))
	for _, q := range p.Questions {
		qs = append(qs, q.Question)
	}
	if len(qs) != uniqueStringCount(qs) {
		return fmt.Errorf("AskUserQuestion: question texts must be unique")
	}
	for _, q := range p.Questions {
		labels := make([]string, 0, len(q.Options))
		for _, o := range q.Options {
			labels = append(labels, o.Label)
		}
		if len(labels) != uniqueStringCount(labels) {
			return fmt.Errorf("AskUserQuestion: option labels must be unique within each question")
		}
	}
	return nil
}

func uniqueStringCount(ss []string) int {
	seen := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		seen[s] = struct{}{}
	}
	return len(seen)
}
