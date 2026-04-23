package localtools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// PowerShellAllowed reports whether local PowerShell execution is enabled.
func PowerShellAllowed() bool {
	return envTruthy("CCB_ENGINE_LOCAL_POWERSHELL")
}

// PowerShellFromJSON runs pwsh (Unix) or powershell.exe (Windows) with -NoProfile -NonInteractive -Command.
// Mirrors TS PowerShellTool subset: timeout ms, no run_in_background. Set CCB_ENGINE_LOCAL_POWERSHELL=1 to allow.
func PowerShellFromJSON(ctx context.Context, raw []byte, workDir string) (string, bool, error) {
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
	if in.RunInBackground != nil && *in.RunInBackground {
		return "", true, fmt.Errorf("run_in_background is not supported in the Go parity runner")
	}
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" {
		return "", true, fmt.Errorf("empty command")
	}
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

	wd := strings.TrimSpace(workDir)
	if wd == "" {
		wd = "."
	}
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

func powershellExeAndArgs(script string) (exe string, args []string) {
	if runtime.GOOS == "windows" {
		return "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script}
	}
	return "pwsh", []string{"-NoProfile", "-NonInteractive", "-Command", script}
}
