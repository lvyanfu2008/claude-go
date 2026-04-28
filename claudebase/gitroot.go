package claudebase

import (
	"os"
	"path/filepath"
	"strings"
)

// FindGitRoot walks up from startPath for .git file or directory (mirrors findGitRootImpl).
func FindGitRoot(startPath string) string {
	cur, err := filepath.Abs(startPath)
	if err != nil {
		return ""
	}
	for {
		gitPath := filepath.Join(cur, ".git")
		if fi, err := os.Lstat(gitPath); err == nil {
			if fi.IsDir() || fi.Mode().IsRegular() {
				return filepath.Clean(cur)
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// ResolveCanonicalGitRoot mirrors src/utils/git.ts resolveCanonicalRoot wrapper output.
func ResolveCanonicalGitRoot(startPath string) string {
	root := FindGitRoot(startPath)
	if root == "" {
		return ""
	}
	return resolveCanonicalGitRoot(root)
}

func resolveCanonicalGitRoot(root string) string {
	gitEntry := filepath.Join(root, ".git")
	fi, err := os.Lstat(gitEntry)
	if err != nil {
		return root
	}
	if fi.IsDir() {
		return filepath.Clean(root)
	}
	data, err := os.ReadFile(gitEntry)
	if err != nil {
		return filepath.Clean(root)
	}
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "gitdir:") {
		return filepath.Clean(root)
	}
	worktreeGitDir, err := filepath.Abs(filepath.Join(root, strings.TrimSpace(strings.TrimPrefix(content, "gitdir:"))))
	if err != nil {
		return filepath.Clean(root)
	}
	commonDirRaw, err := os.ReadFile(filepath.Join(worktreeGitDir, "commondir"))
	if err != nil {
		return filepath.Clean(root)
	}
	commonDir, err := filepath.Abs(filepath.Join(worktreeGitDir, strings.TrimSpace(string(commonDirRaw))))
	if err != nil {
		return filepath.Clean(root)
	}
	// Validate worktreeGitDir is direct child of <commonDir>/worktrees
	worktreesParent := filepath.Dir(worktreeGitDir)
	expected := filepath.Join(commonDir, "worktrees")
	if !strings.EqualFold(filepath.Clean(worktreesParent), filepath.Clean(expected)) {
		return filepath.Clean(root)
	}
	gitdirBack, err := os.ReadFile(filepath.Join(worktreeGitDir, "gitdir"))
	if err != nil {
		return filepath.Clean(root)
	}
	back := strings.TrimSpace(string(gitdirBack))
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	expectedBack, err := filepath.EvalSymlinks(filepath.Join(realRoot, ".git"))
	if err != nil {
		expectedBack = filepath.Join(realRoot, ".git")
	}
	backResolved, err := filepath.EvalSymlinks(back)
	if err != nil {
		backResolved = back
	}
	if !strings.EqualFold(filepath.Clean(backResolved), filepath.Clean(expectedBack)) {
		return filepath.Clean(root)
	}
	base := filepath.Base(commonDir)
	if base != ".git" {
		return filepath.Clean(commonDir)
	}
	return filepath.Clean(filepath.Dir(commonDir))
}
