package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var askUserQuestionAllowedKeys = map[string]struct{}{
	"questions":   {},
	"answers":     {},
	"annotations": {},
	"metadata":    {},
}

type askQuestionItem struct {
	Question    string              `json:"question"`
	Header      string              `json:"header"`
	Options     []askQuestionOption `json:"options"`
	MultiSelect bool                `json:"multiSelect"`
}

type askQuestionOption struct {
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Preview     *string `json:"preview"`
}

func validateAskUserQuestionZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("ask_user_question: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := askUserQuestionAllowedKeys[k]; !ok {
			return fmt.Errorf("ask_user_question: unknown field %q", k)
		}
	}

	qRaw, ok := raw["questions"]
	if !ok {
		return fmt.Errorf("ask_user_question: missing required field %q", "questions")
	}
	var questions []json.RawMessage
	if err := json.Unmarshal(qRaw, &questions); err != nil {
		return fmt.Errorf("ask_user_question: questions must be an array: %w", err)
	}
	if len(questions) < 1 || len(questions) > 4 {
		return fmt.Errorf("ask_user_question: questions must have 1-4 items")
	}
	for i, q := range questions {
		var item askQuestionItem
		if err := json.Unmarshal(q, &item); err != nil {
			return fmt.Errorf("ask_user_question: questions[%d]: %w", i, err)
		}
		if strings.TrimSpace(item.Question) == "" {
			return fmt.Errorf("ask_user_question: questions[%d]: question must be non-empty", i)
		}
		if strings.TrimSpace(item.Header) == "" {
			return fmt.Errorf("ask_user_question: questions[%d]: header must be non-empty", i)
		}
		if len(item.Options) < 2 || len(item.Options) > 4 {
			return fmt.Errorf("ask_user_question: questions[%d]: options must have 2-4 items", i)
		}
		for j, opt := range item.Options {
			if strings.TrimSpace(opt.Label) == "" {
				return fmt.Errorf("ask_user_question: questions[%d].options[%d]: label must be non-empty", i, j)
			}
			if strings.TrimSpace(opt.Description) == "" {
				return fmt.Errorf("ask_user_question: questions[%d].options[%d]: description must be non-empty", i, j)
			}
		}
	}

	// answers: optional object with string values
	if ar, ok := raw["answers"]; ok {
		var ans map[string]any
		if err := json.Unmarshal(ar, &ans); err != nil {
			return fmt.Errorf("ask_user_question: answers: %w", err)
		}
		for k, v := range ans {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("ask_user_question: answers[%s] must be a string", k)
			}
		}
	}

	// annotations: optional object with preview/notes per question
	if ar, ok := raw["annotations"]; ok {
		var ann map[string]json.RawMessage
		if err := json.Unmarshal(ar, &ann); err != nil {
			return fmt.Errorf("ask_user_question: annotations: %w", err)
		}
		for k, v := range ann {
			var entry struct {
				Preview *string `json:"preview"`
				Notes   *string `json:"notes"`
			}
			if err := json.Unmarshal(v, &entry); err != nil {
				return fmt.Errorf("ask_user_question: annotations[%s]: %w", k, err)
			}
		}
	}

	// metadata: optional object with optional source
	if mr, ok := raw["metadata"]; ok {
		var meta struct {
			Source *string `json:"source"`
		}
		if err := json.Unmarshal(mr, &meta); err != nil {
			return fmt.Errorf("ask_user_question: metadata: %w", err)
		}
	}

	return nil
}
