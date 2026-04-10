// Package apilog mirrors TS logLlmApiRequestBody / logLlmApiResponseBody (src/utils/debug.ts)
// when CLAUDE_CODE_LOG_API_REQUEST_BODY / CLAUDE_CODE_LOG_API_RESPONSE_BODY are set.
package apilog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/ccb-engine/debugpath"
	"goc/ccb-engine/settingsfile"
)

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ApiBodyLoggingEnabled is true when either CLAUDE_CODE_LOG_API_REQUEST_BODY or
// CLAUDE_CODE_LOG_API_RESPONSE_BODY is set (same truthy rules as logging).
func ApiBodyLoggingEnabled() bool {
	return envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") || envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY")
}

// ResolvedLogPath returns the file path apilog would use — same as
// goc/ccb-engine/debugpath.ResolveLogPath (mirrors src/utils/debug.ts getDebugLogPath).
func ResolvedLogPath() string {
	return logPath()
}

// MaybePrintDiag prints resolved path and flag state to stderr when CLAUDE_CODE_APILOG_DIAG is truthy.
func MaybePrintDiag() {
	if !envTruthy("CLAUDE_CODE_APILOG_DIAG") {
		return
	}
	path := ResolvedLogPath()
	fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: CLAUDE_CODE_LOG_API_REQUEST_BODY=%v CLAUDE_CODE_LOG_API_RESPONSE_BODY=%v\n",
		envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY"), envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY"))
	fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: resolved log path: %q\n", path)
	if path == "" {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: path empty — check HOME; or set CLAUDE_CODE_DEBUG_LOG_FILE / CLAUDE_CODE_DEBUG_LOGS_DIR\n")
	}
	if !ApiBodyLoggingEnabled() {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: logging flags off → PrepareIfEnabled does nothing → no debug log file created\n")
	}
	if p := settingsfile.UserClaudeSettingsPath(); p != "" {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: user env merge reads %q (override dir: CLAUDE_CONFIG_DIR; project also uses .claude/settings.local.json)\n", p)
	}
	if root := settingsfile.ProjectRootLastResolved(); root != "" {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] diag: project root for Go .claude/settings.go.json (and local): %q (set CCB_ENGINE_PROJECT_ROOT to override)\n", root)
	}
}

// PrepareIfEnabled creates the log file and its parent directories when either
// CLAUDE_CODE_LOG_API_REQUEST_BODY or CLAUDE_CODE_LOG_API_RESPONSE_BODY is truthy,
// and prints the resolved path once on stderr. Call after project .claude/settings.go.json
// (and local) env is applied so CLAUDE_CODE_DEBUG_LOG_* from settings take effect.
//
// Without this, ~/.claude/debug only appears after the first LLM request when logging is on.
func PrepareIfEnabled() {
	if !envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") && !envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY") {
		return
	}
	prepareOnce.Do(prepareLogDestination)
}

var prepareOnce sync.Once

func prepareLogDestination() {
	path := logPath()
	if path == "" {
		warnNoPath.Do(func() {
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] no log path (set HOME, or CLAUDE_CODE_DEBUG_LOG_FILE, or CLAUDE_CODE_DEBUG_LOGS_DIR); API body logs dropped\n")
		})
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] mkdir %q: %v\n", dir, err)
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] open %q: %v\n", path, err)
		return
	}
	_ = f.Close()
	debugpath.MaybeUpdateLatestSymlink(path)
	announcePath.Do(func() {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] writing LLM API bodies to %s\n", path)
	})
}

// LogRequestBody when CLAUDE_CODE_LOG_API_REQUEST_BODY is truthy.
func LogRequestBody(label string, rawJSON []byte) {
	if !envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") {
		return
	}
	writeLog("API_REQUEST_BODY", label, rawJSON)
}

// LogResponseBody when CLAUDE_CODE_LOG_API_RESPONSE_BODY is truthy.
func LogResponseBody(label string, rawJSON []byte) {
	if !envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY") {
		return
	}
	writeLog("API_RESPONSE_BODY", label, rawJSON)
}

var announcePath, warnNoPath sync.Once

func writeLog(kind, label string, raw []byte) {
	serialized := formatJSONForLog(raw)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	out := fmt.Sprintf("%s [%s] %s\n%s\n----------\n", ts, kind, label, serialized)
	path := logPath()
	if path == "" {
		warnNoPath.Do(func() {
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] no log path (set HOME, or CLAUDE_CODE_DEBUG_LOG_FILE, or CLAUDE_CODE_DEBUG_LOGS_DIR); API body logs dropped\n")
		})
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] mkdir %q: %v\n", dir, err)
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] open %q: %v\n", path, err)
		return
	}
	announcePath.Do(func() {
		fmt.Fprintf(os.Stderr, "[ccb-engine apilog] writing LLM API bodies to %s\n", path)
	})
	_, _ = f.WriteString(out)
	_ = f.Close()
	debugpath.MaybeUpdateLatestSymlink(path)
}

func formatJSONForLog(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if json.Compact(&buf, raw) == nil {
		return buf.String()
	}
	return string(raw)
}

func logPath() string {
	return debugpath.ResolveLogPath()
}
