package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func createWorktree(projectRoot, requestedName string) (string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return "", fmt.Errorf("project root is required for worktree isolation")
	}
	name := strings.TrimSpace(requestedName)
	if name == "" {
		name = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	safe := sanitizeName(name)
	base := filepath.Join(root, ".claude", "worktrees")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(base, safe)
	branch := "agent/" + safe
	cmd := exec.Command("git", "-C", root, "worktree", "add", "-b", branch, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create worktree failed: %s", strings.TrimSpace(string(out)))
	}
	return path, nil
}

func removeWorktree(projectRoot, worktreePath string) error {
	root := strings.TrimSpace(projectRoot)
	wp := strings.TrimSpace(worktreePath)
	if root == "" || wp == "" {
		return nil
	}
	cmd := exec.Command("git", "-C", root, "worktree", "remove", "--force", wp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove worktree failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "agent"
	}
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '/' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "agent"
	}
	return out
}
