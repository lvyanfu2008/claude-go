package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AskUserQuestionFromJSON returns synthetic answers when AskAutoFirst is true; otherwise errors (non-interactive policy).
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
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if len(in.Questions) < 1 || len(in.Questions) > 4 {
		return "", true, fmt.Errorf("questions must have 1-4 entries")
	}
	if !c.AskAutoFirst {
		return "", true, fmt.Errorf("AskUserQuestion requires AskAutoFirst (gou-demo enables by default; set GOU_DEMO_NO_ASK_AUTO_FIRST=1 only if you use the TS socket worker for real prompts)")
	}
	answers := map[string]string{}
	for _, q := range in.Questions {
		qt := strings.TrimSpace(q.Question)
		if qt == "" {
			return "", true, fmt.Errorf("empty question text")
		}
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return "", true, fmt.Errorf("each question needs 2-4 options")
		}
		if q.MultiSelect {
			answers[qt] = strings.TrimSpace(q.Options[0].Label)
		} else {
			answers[qt] = strings.TrimSpace(q.Options[0].Label)
		}
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
