package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TasksResult is the JSON payload for /tasks.
type TasksResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleTasksCommand handles /tasks.
func HandleTasksCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()
	tasksDir := filepath.Join(cwd, ".harness", ".gou-tasks")

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return json.Marshal(TasksResult{
			Type: "text",
			Value: fmt.Sprintf("No active background tasks.\nTask directory: %s", tasksDir),
		})
	}

	var lines []string
	lines = append(lines, "Active background tasks:")
	for _, entry := range entries {
		if entry.IsDir() {
			lines = append(lines, fmt.Sprintf("  - %s", entry.Name()))
		}
	}
	if len(lines) == 1 {
		return json.Marshal(TasksResult{
			Type: "text", Value: "No active background tasks.",
		})
	}
	return json.Marshal(TasksResult{
		Type: "text", Value: strings.Join(lines, "\n"),
	})
}
