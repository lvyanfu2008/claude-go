package paritytools

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

func trimProjectRoot(root string) string {
	return strings.TrimSpace(root)
}

// TaskBaseDir returns the directory used for TaskOutput / TaskStop file protocol
// (under project .claude, scoped by session).
func (c Config) TaskBaseDir() string {
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	sid := strings.TrimSpace(c.SessionID)
	if sid == "" {
		sid = "default-session"
	}
	return filepath.Join(pr, ".claude", ".gou-tasks", sid)
}

func (c Config) TasksDir() string {
	return filepath.Join(c.TaskBaseDir(), "tasks")
}

// TodoFilePath persists TodoWrite state (process-local parity with TS appState.todos).
func (c Config) TodoFilePath() string {
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	sid := strings.TrimSpace(c.SessionID)
	if sid == "" {
		sid = "default-session"
	}
	return filepath.Join(pr, ".claude", "gou_demo_todos_"+sanitizeFilePart(sid)+".json")
}

func (c Config) PlanModePath() string {
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	return filepath.Join(pr, ".claude", "gou_plan_mode.json")
}

func sanitizeFilePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}
