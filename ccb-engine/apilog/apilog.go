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
	"sync/atomic"
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

// RequestBodyLoggingEnabled is true when CLAUDE_CODE_LOG_API_REQUEST_BODY is truthy.
func RequestBodyLoggingEnabled() bool {
	return envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY")
}

// ResponseBodyLoggingEnabled is true when CLAUDE_CODE_LOG_API_RESPONSE_BODY is truthy.
func ResponseBodyLoggingEnabled() bool {
	return envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY")
}

// ResolvedLogPath returns the file path apilog would use — same as
// goc/ccb-engine/debugpath.ResolveLogPath (mirrors src/utils/debug.ts getDebugLogPath).
func ResolvedLogPath() string {
	return logPath()
}

// MaybePrintDiag records resolved path and flag state when CLAUDE_CODE_APILOG_DIAG is truthy.
// It appends to the same file as [ResolvedLogPath] / LLM API body logs (~/.claude/debug/<session>.txt
// by default) so TTY sessions are not spammed; if the path is empty or the write fails, it falls back to stderr.
func MaybePrintDiag() {
	if !envTruthy("CLAUDE_CODE_APILOG_DIAG") {
		return
	}
	path := ResolvedLogPath()
	var b strings.Builder
	fmt.Fprintf(&b, "[ccb-engine apilog] diag: CLAUDE_CODE_LOG_API_REQUEST_BODY=%v CLAUDE_CODE_LOG_API_RESPONSE_BODY=%v\n",
		envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY"), envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY"))
	fmt.Fprintf(&b, "[ccb-engine apilog] diag: resolved log path: %q\n", path)
	if lp := debugpath.LatestLinkPathFor(path); lp != "" {
		fmt.Fprintf(&b, "[ccb-engine apilog] diag: latest symlink (same log): %q\n", lp)
	}
	if path == "" {
		fmt.Fprintf(&b, "[ccb-engine apilog] diag: path empty — check HOME; or set CLAUDE_CODE_DEBUG_LOG_FILE / CLAUDE_CODE_DEBUG_LOGS_DIR\n")
	}
	if !ApiBodyLoggingEnabled() {
		fmt.Fprintf(&b, "[ccb-engine apilog] diag: logging flags off → PrepareIfEnabled does nothing → no debug log file created\n")
	}
	if p := settingsfile.UserClaudeSettingsPath(); p != "" {
		fmt.Fprintf(&b, "[ccb-engine apilog] diag: user env merge reads %q (override dir: CLAUDE_CONFIG_DIR; project also uses .claude/settings.local.json)\n", p)
	}
	if root := settingsfile.ProjectRootLastResolved(); root != "" {
		fmt.Fprintf(&b, "[ccb-engine apilog] diag: project root for Go .claude/settings.go.json (and local): %q (set CCB_ENGINE_PROJECT_ROOT to override)\n", root)
	}
	body := b.String()
	if path != "" {
		if err := appendDiagToLog(path, body); err == nil {
			debugpath.MaybeUpdateLatestSymlink(path)
			return
		}
	}
	_, _ = fmt.Fprint(os.Stderr, body)
}

func appendDiagToLog(path, body string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = fmt.Fprintf(f, "%s [APILOG_DIAG]\n%s", ts, body)
	return err
}

// PrepareIfEnabled creates the log file and its parent directories when either
// CLAUDE_CODE_LOG_API_REQUEST_BODY or CLAUDE_CODE_LOG_API_RESPONSE_BODY is truthy,
// and writes a one-time "latest points to …" line to that log file (not stderr). Call after project .claude/settings.go.json
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

// Monotonic per-process counters; tags are concatenated to the body line (llmRequest-3{…}).
var llmRequestSeq, llmResponseSeq atomic.Uint64

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
		if err := appendApilogPathAnnounce(path); err != nil {
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] LLM API bodies: %s points to %s\n", debugpath.LatestLinkPathFor(path), path)
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] (could not write announce line to log: %v)\n", err)
		}
	})
}

// LogRequestBody when CLAUDE_CODE_LOG_API_REQUEST_BODY is truthy.
// Prefixes the serialized body line with llmRequest-N (no space) for grep.
func LogRequestBody(label string, rawJSON []byte) {
	if !envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") {
		return
	}
	n := llmRequestSeq.Add(1)
	writeLog("API_REQUEST_BODY", label, fmt.Sprintf("llmRequest-%d", n), rawJSON)
}

// LogResponseBody when CLAUDE_CODE_LOG_API_RESPONSE_BODY is truthy.
// Prefixes the serialized body line with llmResponse-N (no space) for grep.
func LogResponseBody(label string, rawJSON []byte) {
	if !envTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY") {
		return
	}
	n := llmResponseSeq.Add(1)
	writeLog("API_RESPONSE_BODY", label, fmt.Sprintf("llmResponse-%d", n), rawJSON)
}

var announcePath, warnNoPath sync.Once

// appendApilogPathAnnounce writes the one-time "latest points to …" line to the log file (not stderr).
func appendApilogPathAnnounce(path string) error {
	if path == "" {
		return fmt.Errorf("empty log path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	lp := debugpath.LatestLinkPathFor(path)
	_, err = fmt.Fprintf(f, "%s [ccb-engine apilog] LLM API bodies: %s points to %s\n", ts, lp, path)
	return err
}

func writeLog(kind, label, bodyPrefix string, raw []byte) {
	serialized := formatJSONForLog(raw)
	bodyOut := joinBodyPrefix(bodyPrefix, serialized)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	out := fmt.Sprintf("%s [%s] %s\n%s\n----------\n", ts, kind, label, bodyOut)
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
		if err := appendApilogPathAnnounce(path); err != nil {
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] LLM API bodies: %s points to %s\n", debugpath.LatestLinkPathFor(path), path)
			fmt.Fprintf(os.Stderr, "[ccb-engine apilog] (could not write announce line to log: %v)\n", err)
		}
	})
	_, _ = f.WriteString(out)
	_ = f.Close()
	debugpath.MaybeUpdateLatestSymlink(path)
}

func joinBodyPrefix(prefix, serialized string) string {
	if prefix == "" {
		return serialized
	}
	if serialized == "" {
		return prefix
	}
	return prefix + serialized
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
