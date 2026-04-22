package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type todoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

// TodoWriteFromJSON persists todos next to the project .claude tree (subset parity with TS TodoWriteTool).
func TodoWriteFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		Todos []todoItem `json:"todos"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	path := c.TodoFilePath()
	var old []todoItem
	if prev, err := readFileLimited(path, 1<<20); err == nil {
		_ = json.Unmarshal(prev, &old)
	}
	for _, t := range in.Todos {
		if strings.TrimSpace(t.Content) == "" {
			return "", true, fmt.Errorf("todo content cannot be empty")
		}
		if strings.TrimSpace(t.ActiveForm) == "" {
			return "", true, fmt.Errorf("todo activeForm cannot be empty")
		}
		switch t.Status {
		case "pending", "in_progress", "completed":
		default:
			return "", true, fmt.Errorf("invalid todo status %q", t.Status)
		}
	}
	allDone := true
	for _, t := range in.Todos {
		if t.Status != "completed" {
			allDone = false
			break
		}
	}
	stored := in.Todos
	if allDone {
		stored = nil
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return "", true, err
	}
	out := map[string]any{
		"data": map[string]any{
			"oldTodos":                old,
			"newTodos":                in.Todos,
			"verificationNudgeNeeded": false,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
