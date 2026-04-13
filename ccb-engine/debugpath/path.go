// Package debugpath mirrors src/utils/debug.ts getDebugLogPath / session debug file layout.
package debugpath

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	sessionOnce sync.Once
	sessionID   string
)

// SessionID is a stable per-process identifier (32 hex chars), analogous to TS session-scoped debug files.
func SessionID() string {
	sessionOnce.Do(func() {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			sessionID = "fallback-session-id"
			return
		}
		sessionID = hex.EncodeToString(b[:])
	})
	return sessionID
}

// ConfigHomeDir matches getClaudeConfigHomeDir: CLAUDE_CONFIG_DIR, else ~/.claude.
func ConfigHomeDir() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return d
	}
	h, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(h) == "" {
		return ""
	}
	return filepath.Join(h, ".claude")
}

// ResolveLogPath matches src/utils/debug.ts getDebugLogPath resolution order:
//  1. CLAUDE_CODE_DEBUG_LOG_FILE — explicit file
//  2. CLAUDE_CODE_DEBUG_LOGS_DIR — if an existing directory (or trailing path sep / no extension path), <dir>/<sessionId>.txt; otherwise the value is used as the full file path (TS passes through). Non-absolute values are joined under ConfigHomeDir() (CLAUDE_CONFIG_DIR or ~/.claude), not the process cwd.
//  3. <configHome>/debug/<sessionId>.txt
func ResolveLogPath() string {
	if p := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DEBUG_LOG_FILE")); p != "" {
		return p
	}
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DEBUG_LOGS_DIR")); d != "" {
		return resolveDebugLogsDir(d)
	}
	cfg := ConfigHomeDir()
	if cfg == "" {
		return ""
	}
	return filepath.Join(cfg, "debug", SessionID()+".txt")
}

func resolveDebugLogsDir(d string) string {
	d = strings.TrimSpace(d)
	if d == "" {
		return ""
	}
	if !filepath.IsAbs(d) {
		if cfg := ConfigHomeDir(); cfg != "" {
			d = filepath.Join(cfg, d)
		}
	}
	if fi, err := os.Stat(d); err == nil {
		if fi.IsDir() {
			return filepath.Join(d, SessionID()+".txt")
		}
		return d
	}
	if strings.HasSuffix(d, string(os.PathSeparator)) {
		return filepath.Join(strings.TrimSuffix(d, string(os.PathSeparator)), SessionID()+".txt")
	}
	if filepath.Ext(d) == "" {
		return filepath.Join(d, SessionID()+".txt")
	}
	return d
}

// LatestLinkPathFor returns filepath.Join(filepath.Dir(logPath), "latest").
// It is the symlink path [MaybeUpdateLatestSymlink] updates to point at the
// current log file (same as TS join(dirname(getDebugLogPath()), 'latest')).
func LatestLinkPathFor(logPath string) string {
	if logPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(logPath), "latest")
}

// MaybeUpdateLatestSymlink creates <debug-dir>/latest -> logPath (best-effort; TS parity:
// ~/.claude/debug/latest — see src/utils/debug.ts updateLatestDebugLogSymlink).
func MaybeUpdateLatestSymlink(logPath string) {
	if logPath == "" {
		return
	}
	abs, err := filepath.Abs(logPath)
	target := logPath
	if err == nil && abs != "" {
		target = abs
	}
	latest := LatestLinkPathFor(logPath)
	_ = os.Remove(latest)
	_ = os.Symlink(target, latest)
}
