package hookexec

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DefaultHookTimeoutMs mirrors TOOL_HOOK_EXECUTION_TIMEOUT_MS in hooks.ts (10 minutes).
const DefaultHookTimeoutMs = 10 * 60 * 1000

type commandHook struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Timeout *float64 `json:"timeout"` // seconds in settings JSON
}

func hookTimeoutMS(h commandHook, batchDefault int) int {
	if h.Timeout != nil && *h.Timeout > 0 {
		ms := int(*h.Timeout * 1000)
		if ms > 30*60*1000 {
			return 30 * 60 * 1000
		}
		if ms < 1000 {
			return 1000
		}
		return ms
	}
	if batchDefault > 0 {
		return batchDefault
	}
	return DefaultHookTimeoutMs
}

// RunCommandHook runs a single command hook: sh -c with jsonInput written to stdin (plus newline), like TS execCommandHook.
func RunCommandHook(ctx context.Context, workDir, command, jsonInput string, timeoutMs int, extraEnv []string) (stdout, stderr string, exitCode int, err error) {
	if strings.TrimSpace(command) == "" {
		return "", "", 0, nil
	}
	if timeoutMs <= 0 {
		timeoutMs = DefaultHookTimeoutMs
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(cctx, "/bin/sh", "-c", command)
	}
	cmd.Dir = strings.TrimSpace(workDir)
	if cmd.Dir == "" {
		cmd.Dir = "."
	}
	env := os.Environ()
	if len(extraEnv) > 0 {
		env = append(env, extraEnv...)
	}
	// If the command lives under a plugin cache directory, set CLAUDE_PLUGIN_ROOT
	// so hook scripts can detect the plugin environment.
	if pluginRoot := extractPluginRoot(command); pluginRoot != "" {
		env = append(env, "CLAUDE_PLUGIN_ROOT="+pluginRoot)
	}
	cmd.Env = env
	cmd.Stdin = strings.NewReader(jsonInput + "\n")
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = strings.TrimSpace(outBuf.String())
	stderr = strings.TrimSpace(errBuf.String())
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			return stdout, stderr, ee.ExitCode(), runErr
		}
		if cctx.Err() == context.DeadlineExceeded || cctx.Err() == context.Canceled {
			return stdout, stderr, -1, cctx.Err()
		}
		return stdout, stderr, -1, runErr
	}
	return stdout, stderr, 0, nil
}

// ParseHookJSONOutput extracts additionalContext from hook stdout.
// Handles both TS nested format {"hookSpecificOutput":{"additionalContext":"..."}}
// and plain format {"additionalContext":"..."} (used when CLAUDE_PLUGIN_ROOT is unset).
func ParseHookJSONOutput(stdout, expectedEvent string) (additionalContext string, _ error) {
	s := strings.TrimSpace(stdout)
	if s == "" {
		return "", nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &top); err != nil {
		return "", nil
	}

	// Try nested hookSpecificOutput first (TS Claude Code format).
	if rawHSO, ok := top["hookSpecificOutput"]; ok && len(rawHSO) > 0 {
		var hso struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		}
		if err := json.Unmarshal(rawHSO, &hso); err != nil {
			return "", nil
		}
		if expectedEvent != "" && hso.HookEventName != "" && hso.HookEventName != expectedEvent {
			return "", nil
		}
		return strings.TrimSpace(hso.AdditionalContext), nil
	}

	// Fallback: top-level additionalContext (plain format).
	if rawAC, ok := top["additionalContext"]; ok && len(rawAC) > 0 {
		var ac string
		if err := json.Unmarshal(rawAC, &ac); err != nil {
			return "", nil
		}
		return strings.TrimSpace(ac), nil
	}
	return "", nil
}

// extractPluginRoot returns the plugin root directory if the command path
// is inside a plugin cache directory, otherwise returns empty string.
func extractPluginRoot(command string) string {
	// The command contains a path like:
	// .../plugins/cache/{marketplace}/{name}/{version}/hooks/script
	// Extract up to the version directory.
	fields := strings.Fields(command)
	for _, f := range fields {
		f = strings.Trim(f, `"'`)
		f = strings.ReplaceAll(f, "\\", "/") // normalize Windows backslashes
		idx := strings.Index(f, "/.claude/plugins/cache/")
		if idx < 0 {
			continue
		}
		rest := f[idx+len("/.claude/plugins/cache/"):]
		parts := strings.Split(rest, "/")
		if len(parts) < 3 {
			continue
		}
		// marketplace/name/version
		return f[:idx] + "/.claude/plugins/cache/" + strings.Join(parts[:3], "/")
	}
	return ""
}
