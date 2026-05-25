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
	base := filepath.Join(root, ".harness", "worktrees")
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

// CleanupStaleAgentWorktrees removes worktrees whose last modification time exceeds maxAge.
// Mirrors TS cleanupStaleAgentWorktrees in worktree.ts.
func CleanupStaleAgentWorktrees(projectRoot string, maxAge time.Duration) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return
	}
	base := filepath.Join(root, ".harness", "worktrees")
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		wp := filepath.Join(base, ent.Name())
		info, err := ent.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		// Verify it's a git worktree before removing.
		if !isGitWorktree(root, wp) {
			continue
		}
		if HasWorktreeChanges(wp) {
			continue
		}
		_ = removeWorktree(root, wp)
	}
}

// HasWorktreeChanges checks if a worktree has uncommitted changes.
// Mirrors TS hasWorktreeChanges in worktree.ts.
func HasWorktreeChanges(worktreePath string) bool {
	wp := strings.TrimSpace(worktreePath)
	if wp == "" {
		return false
	}
	cmd := exec.Command("git", "-C", wp, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return true // err on side of caution
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// BumpWorktreeMtime touches a marker file to update the worktree's effective age.
// Mirrors TS bumpWorktreeMtime in worktree.ts.
func BumpWorktreeMtime(worktreePath string) {
	wp := strings.TrimSpace(worktreePath)
	if wp == "" {
		return
	}
	marker := filepath.Join(wp, ".harness", "worktree-marker")
	_ = os.MkdirAll(filepath.Dir(marker), 0o700)
	_ = os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339)), 0o600)
}

func isGitWorktree(projectRoot, worktreePath string) bool {
	cmd := exec.Command("git", "-C", projectRoot, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), worktreePath)
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
