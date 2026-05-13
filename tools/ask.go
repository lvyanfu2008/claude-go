package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AskUserQuestionFromJSON returns answers: from interactive user input when answers are already
// in the tool input; from auto-first selection when AskAutoFirst is true; otherwise errors.
func AskUserQuestionFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		Questions []struct {
			Question    string `json:"question"`
			Header      string `json:"header"`
			MultiSelect bool   `json:"multiSelect"`
			Options     []struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"options"`
		} `json:"questions"`
		Answers     map[string]string `json:"answers"`
		Annotations map[string]any    `json:"annotations"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if len(in.Questions) < 1 || len(in.Questions) > 4 {
		return "", true, fmt.Errorf("questions must have 1-4 entries")
	}

	// Interactive mode: answers were filled in by the question UI and merged into updatedInput.
	if len(in.Answers) > 0 {
		out := map[string]any{
			"data": map[string]any{
				"questions":   in.Questions,
				"answers":     in.Answers,
				"annotations": in.Annotations,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	if !c.AskAutoFirst {
		return "", true, fmt.Errorf("AskUserQuestion requires AskAutoFirst (gou-demo enables by default; set GOU_DEMO_NO_ASK_AUTO_FIRST=1 only if you use the TS socket worker for real prompts)")
	}

	// Auto-first mode: pick first option for each question.
	answers := map[string]string{}
	for _, q := range in.Questions {
		qt := strings.TrimSpace(q.Question)
		if qt == "" {
			return "", true, fmt.Errorf("empty question text")
		}
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return "", true, fmt.Errorf("each question needs 2-4 options")
		}
		answers[qt] = strings.TrimSpace(q.Options[0].Label)
	}
	out := map[string]any{
		"data": map[string]any{
			"questions": in.Questions,
			"answers":   answers,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
