// Package diaglog appends one-off diagnostic lines (command load, tool load) to the Claude debug log file,
// matching TS getDebugLogPath resolution — not stderr, so full-screen TUI (gou-demo) is not corrupted.
// Use [LineOrStderr] when the line should still appear on stderr if no log file path is resolved.
package diaglog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/ccb-engine/debugpath"
)

// Line appends a single line to the diagnostic log. Uses CLAUDE_CODE_DIAG_LOG_FILE if set, else [debugpath.ResolveLogPath]. If the resolved path is empty, drops the line.
func Line(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	path := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DIAG_LOG_FILE"))
	if path == "" {
		path = debugpath.ResolveLogPath()
	}
	if path == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = fmt.Fprintf(f, "%s %s", ts, msg)
}

// LineOrStderr is like [Line] but if no log file path is resolved, writes the same
// line to [os.Stderr] instead of dropping it (use for diagnostics that should not be silent).
func LineOrStderr(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	path := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DIAG_LOG_FILE"))
	if path == "" {
		path = debugpath.ResolveLogPath()
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	out := fmt.Sprintf("%s %s", ts, msg)
	if path == "" {
		_, _ = os.Stderr.WriteString(out)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		_, _ = os.Stderr.WriteString(out)
		return
	}
	defer f.Close()
	_, _ = f.WriteString(out)
}

var mu sync.Mutex
