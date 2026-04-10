// Package diaglog appends one-off diagnostic lines (command load, tool load) to the Claude debug log file,
// matching TS getDebugLogPath resolution — not stderr, so full-screen TUI (gou-demo) is not corrupted.
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

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// Line appends a single line to the diagnostic log. When CCB_ENGINE_DIAG_TO_STDERR=1, writes to stderr instead.
// Otherwise uses CLAUDE_CODE_DIAG_LOG_FILE if set, else [debugpath.ResolveLogPath]. If the resolved path is empty, drops the line.
func Line(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	if envTruthy("CCB_ENGINE_DIAG_TO_STDERR") {
		_, _ = os.Stderr.WriteString(msg)
		return
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

var mu sync.Mutex
