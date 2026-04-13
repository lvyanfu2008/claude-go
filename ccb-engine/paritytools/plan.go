package paritytools

import (
	"encoding/json"
	"path/filepath"
	"time"
)

type planModeFile struct {
	Active    bool   `json:"active"`
	EnteredAt string `json:"enteredAt,omitempty"`
}

// EnterPlanModeFromJSON marks plan mode active in `.claude/gou_plan_mode.json`.
func EnterPlanModeFromJSON(_ []byte, c Config) (string, bool, error) {
	path := c.PlanModePath()
	rec := planModeFile{Active: true, EnteredAt: time.Now().UTC().Format(time.RFC3339)}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := ensureDirFromFile(path); err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return "", true, err
	}
	msg := "Entered plan mode. You should now focus on exploring the codebase and designing an implementation approach."
	out := map[string]any{"data": map[string]any{"message": msg}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func ensureDirFromFile(path string) error {
	return ensureDir(filepath.Dir(path))
}

// ExitPlanModeFromJSON clears plan mode; stores optional allowed_prompts in the JSON file for debugging.
func ExitPlanModeFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		AllowedPrompts []struct {
			Tool   string `json:"tool"`
			Prompt string `json:"prompt"`
		} `json:"allowedPrompts"`
	}
	_ = json.Unmarshal(raw, &in)
	path := c.PlanModePath()
	rec := map[string]any{
		"active":         false,
		"exitedAt":       time.Now().UTC().Format(time.RFC3339),
		"allowedPrompts": in.AllowedPrompts,
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := ensureDirFromFile(path); err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return "", true, err
	}
	out := map[string]any{"data": map[string]any{"message": "Exited plan mode."}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
