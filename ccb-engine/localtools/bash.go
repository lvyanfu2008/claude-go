package localtools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BashAllowed reports whether Bash may run. When localDefault is true (e.g. gou-demo ParityToolRunner),
// Bash is on unless GOU_DEMO_NO_LOCAL_BASH or CCB_ENGINE_DISABLE_LOCAL_BASH is set. Otherwise the historical
// gate CCB_ENGINE_LOCAL_BASH=1 applies.
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
func BashFromJSON(ctx context.Context, raw []byte, workDir string, localDefault bool) (string, bool, error) {
	if !BashAllowed(localDefault) {
		return "", true, fmt.Errorf("Bash tool disabled in Go runner (set CCB_ENGINE_LOCAL_BASH=1, or run gou-demo with local Bash default on and without GOU_DEMO_NO_LOCAL_BASH; use socket worker for full TS execution)")
	}
	var in struct {
		Command           string  `json:"command"`
		Timeout           float64 `json:"timeout"`
		Description       string  `json:"description"`
		RunInBackground   *bool   `json:"run_in_background"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if in.RunInBackground != nil && *in.RunInBackground {
		return "", true, fmt.Errorf("run_in_background is not supported in the Go parity runner; omit run_in_background or use the TS socket worker")
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
	var cancel context.CancelFunc
	if cctx == nil {
		cctx = context.Background()
	}
	cctx, cancel = context.WithTimeout(cctx, d)
	defer cancel()

	wd := strings.TrimSpace(workDir)
	if wd == "" {
		wd = "."
	}
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

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
