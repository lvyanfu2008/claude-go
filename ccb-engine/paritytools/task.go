package paritytools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxTaskOutputRead = 8 << 20 // 8MB, matches TS DEFAULT_MAX_READ_BYTES

// TaskOutputFromJSON reads `<tasksDir>/<task_id>.output` (file-protocol parity with TS disk output).
func TaskOutputFromJSON(ctx context.Context, raw []byte, c Config) (string, bool, error) {
	_ = ctx
	var in struct {
		TaskID  string `json:"task_id"`
		Block   *bool  `json:"block"`
		Timeout float64 `json:"timeout"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	id := strings.TrimSpace(in.TaskID)
	if id == "" {
		return "", true, fmt.Errorf("task_id is required")
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return "", true, fmt.Errorf("invalid task_id")
	}
	tasksDir := c.TasksDir()
	if err := ensureDir(tasksDir); err != nil {
		return "", true, err
	}
	outPath := filepath.Join(tasksDir, id+".output")
	stopPath := filepath.Join(tasksDir, id+".stop")

	block := true
	if in.Block != nil {
		block = *in.Block
	}
	timeoutMs := in.Timeout
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if timeoutMs > 600000 {
		timeoutMs = 600000
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return "", true, ctx.Err()
		default:
		}
		data, err := readFileLimited(outPath, maxTaskOutputRead)
		if err == nil {
			resp := map[string]any{
				"retrieval_status": "success",
				"task": map[string]any{
					"task_id":     id,
					"task_type":   "unknown",
					"status":      "unknown",
					"description": "",
					"output":      string(data),
				},
			}
			b, _ := json.Marshal(resp)
			return string(b), false, nil
		}
		if !os.IsNotExist(err) {
			return "", true, err
		}
		if !block {
			resp := map[string]any{
				"retrieval_status": "not_ready",
				"task":             nil,
			}
			b, _ := json.Marshal(resp)
			return string(b), false, nil
		}
		if time.Now().After(deadline) {
			resp := map[string]any{
				"retrieval_status": "timeout",
				"task":             nil,
			}
			b, _ := json.Marshal(resp)
			return string(b), false, nil
		}
		if _, err := os.Stat(stopPath); err == nil {
			resp := map[string]any{
				"retrieval_status": "not_ready",
				"task":             nil,
			}
			b, _ := json.Marshal(resp)
			return string(b), false, nil
		}
		time.Sleep(80 * time.Millisecond)
	}
}

// TaskStopFromJSON writes a stop sentinel file (best-effort; no process kill in Go runner).
func TaskStopFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		TaskID  string `json:"task_id"`
		ShellID string `json:"shell_id"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	id := strings.TrimSpace(in.TaskID)
	if id == "" {
		id = strings.TrimSpace(in.ShellID)
	}
	if id == "" {
		return "", true, fmt.Errorf("missing required parameter: task_id")
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return "", true, fmt.Errorf("invalid task_id")
	}
	tasksDir := c.TasksDir()
	if err := ensureDir(tasksDir); err != nil {
		return "", true, err
	}
	stopPath := filepath.Join(tasksDir, id+".stop")
	f, err := os.OpenFile(stopPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", true, err
	}
	_ = f.Close()
	resp := map[string]any{
		"message":   "Stop signal written (Go runner does not manage live tasks; use TS worker for real task kill)",
		"task_id":   id,
		"task_type": "file_stub",
	}
	b, _ := json.Marshal(resp)
	return string(b), false, nil
}
