package localtools

import (
	"context"
	"encoding/json"
	"fmt"
	"goc/tools/procregistry"
	"goc/tools/tool"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BashAllowed reports whether Bash may run. When localDefault is true (e.g. gou-demo ParityToolRunner),
// Bash is on unless GOU_DEMO_NO_LOCAL_BASH or CCB_ENGINE_DISABLE_LOCAL_BASH is set. Otherwise
// CCB_ENGINE_LOCAL_BASH=1 is required to allow Bash.
func BashAllowed(localDefault bool) bool {
	if envTruthy("GOU_DEMO_NO_LOCAL_BASH") || envTruthy("CCB_ENGINE_DISABLE_LOCAL_BASH") {
		return false
	}
	if envTruthy("CCB_ENGINE_LOCAL_BASH") {
		return true
	}
	return localDefault
}

// BashFromJSON runs a shell command when [BashAllowed] is true for the given default.
// tasksDir is the directory for background-task output/status/stop files.
// When empty, run_in_background is rejected.
func BashFromJSON(ctx context.Context, raw []byte, workDir string, localDefault bool, tasksDir string) (string, bool, error) {
	if !BashAllowed(localDefault) {
		return "", true, fmt.Errorf("Bash tool disabled in Go runner (set CCB_ENGINE_LOCAL_BASH=1, or run gou-demo with local Bash default on and without GOU_DEMO_NO_LOCAL_BASH; use socket worker for full TS execution)")
	}
	var in struct {
		Command         string  `json:"command"`
		Timeout         float64 `json:"timeout"`
		Description     string  `json:"description"`
		RunInBackground *bool   `json:"run_in_background"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" {
		return "", true, fmt.Errorf("empty command")
	}

	wd := strings.TrimSpace(workDir)
	if wd == "" {
		wd = "."
	}

	// Background path.
	if in.RunInBackground != nil && *in.RunInBackground {
		if tasksDir == "" {
			return "", true, fmt.Errorf("run_in_background is not supported in the Go parity runner; omit run_in_background or use the TS socket worker")
		}
		return runBashBackground(cmd, wd, tasksDir)
	}

	// Synchronous path.
	ms := int(in.Timeout)
	if ms <= 0 {
		ms = 120_000
	}
	d := time.Duration(ms) * time.Millisecond
	if d > 30*time.Minute {
		d = 30 * time.Minute
	}
	cctx := ctx
	var cancel context.CancelFunc
	if cctx == nil {
		cctx = context.Background()
	}
	cctx, cancel = context.WithTimeout(cctx, d)
	defer cancel()

	//nolint:gosec // Gated by CCB_ENGINE_LOCAL_BASH; user explicitly enables local shell execution.
	ex := exec.CommandContext(cctx, "sh", "-c", cmd)
	ex.Dir = wd
	ex.Env = os.Environ()
	out, err := ex.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s != "" {
			return s, true, nil
		}
		return "", true, err
	}
	if s == "" {
		return "(no output)", false, nil
	}
	return s, false, nil
}

func runBashBackground(command, workDir, tasksDir string) (string, bool, error) {
	taskID := fmt.Sprintf("bash-%d", time.Now().UTC().UnixNano())
	outputFile := filepath.Join(tasksDir, taskID+".output")

	writeBackgroundStatus(tasksDir, taskID, "running", "Command started", true)

	//nolint:gosec // Gated by CCB_ENGINE_LOCAL_BASH.
	ex := exec.Command("sh", "-c", command)
	ex.Dir = workDir
	ex.Env = os.Environ()
	procregistry.SetProcessGroup(ex)
	procregistry.StoreProcess(taskID, ex)

	go func() {
		defer procregistry.RemoveProcess(taskID)
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			writeBackgroundStatus(tasksDir, taskID, "failed", "Failed to open output file: "+err.Error(), false)
			return
		}
		defer f.Close()
		ex.Stdout = f
		ex.Stderr = f
		if err := ex.Run(); err != nil {
			if isTaskStopRequested(tasksDir, taskID) {
				writeBackgroundStatus(tasksDir, taskID, "stopped", "Task was stopped", false)
				return
			}
			writeBackgroundStatus(tasksDir, taskID, "failed", err.Error(), false)
			return
		}
		writeBackgroundStatus(tasksDir, taskID, "completed", "Command completed", true)
	}()

	out := map[string]any{
		"data": map[string]any{
			"taskId":     taskID,
			"outputFile": outputFile,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// writeBackgroundStatus writes a JSON status file for a background task.
func writeBackgroundStatus(tasksDir, taskID, status, message string, success bool) {
	if tasksDir == "" || taskID == "" {
		return
	}
	_ = os.MkdirAll(tasksDir, 0o700)
	payload := map[string]any{
		"task_id":   taskID,
		"status":    status,
		"success":   success,
		"message":   message,
		"updatedAt": time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(tasksDir, taskID+".status.json"), append(b, '\n'), 0o600)
}

// isTaskStopRequested checks if a .stop sentinel file exists for the given task.
func isTaskStopRequested(tasksDir, taskID string) bool {
	if tasksDir == "" || taskID == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(tasksDir, taskID+".stop"))
	return err == nil
}

func envTruthy(k string) bool {
	return tool.EnvTruthy(k)
}
