package localtools

import (
	"context"
	"encoding/json"
	"fmt"
	"goc/tools/procregistry"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PowerShellAllowed reports whether local PowerShell execution is enabled.
func PowerShellAllowed() bool {
	return envTruthy("CCB_ENGINE_LOCAL_POWERSHELL")
}

// PowerShellFromJSON runs pwsh (Unix) or powershell.exe (Windows) with -NoProfile -NonInteractive -Command.
// Mirrors TS PowerShellTool subset: timeout ms, run_in_background support.
// Set CCB_ENGINE_LOCAL_POWERSHELL=1 to allow.
// tasksDir is the directory for background-task output/status/stop files.
func PowerShellFromJSON(ctx context.Context, raw []byte, workDir string, tasksDir string) (string, bool, error) {
	if !PowerShellAllowed() {
		return "", true, fmt.Errorf("PowerShell disabled in Go runner (set CCB_ENGINE_LOCAL_POWERSHELL=1)")
	}
	var in struct {
		Command         string  `json:"command"`
		Timeout         float64 `json:"timeout"`
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
			return "", true, fmt.Errorf("run_in_background is not supported in the Go parity runner")
		}
		return runPowerShellBackground(cmd, wd, tasksDir)
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
	if cctx == nil {
		cctx = context.Background()
	}
	var cancel context.CancelFunc
	cctx, cancel = context.WithTimeout(cctx, d)
	defer cancel()

	exe, args := powershellExeAndArgs(cmd)
	//nolint:gosec // Gated by CCB_ENGINE_LOCAL_POWERSHELL.
	ex := exec.CommandContext(cctx, exe, args...)
	ex.Dir = wd
	ex.Env = os.Environ()
	combined, err := ex.CombinedOutput()
	s := strings.TrimSpace(string(combined))
	interrupted := cctx.Err() == context.DeadlineExceeded || cctx.Err() == context.Canceled
	data := map[string]any{
		"stdout":      s,
		"stderr":      "",
		"interrupted": interrupted,
	}
	b, _ := json.Marshal(map[string]any{"data": data})
	out := string(b)
	if err != nil {
		if s != "" {
			return out, true, nil
		}
		return "", true, err
	}
	return out, false, nil
}

func runPowerShellBackground(command, workDir, tasksDir string) (string, bool, error) {
	taskID := fmt.Sprintf("ps-%d", time.Now().UTC().UnixNano())
	outputFile := filepath.Join(tasksDir, taskID+".output")

	writeBackgroundStatus(tasksDir, taskID, "running", "PowerShell command started", true)

	exe, args := powershellExeAndArgs(command)
	//nolint:gosec // Gated by CCB_ENGINE_LOCAL_POWERSHELL.
	ex := exec.Command(exe, args...)
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

func powershellExeAndArgs(script string) (exe string, args []string) {
	if runtime.GOOS == "windows" {
		return "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}
	}
	return "pwsh", []string{"-NoProfile", "-NonInteractive", "-Command", script}
}
