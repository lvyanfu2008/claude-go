package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// normalizePathForComparison mirrors src/utils/file.ts normalizePathForComparison (Windows: case + backslash).
func normalizePathForComparison(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(strings.ReplaceAll(p, "/", `\`))
	}
	return p
}

// findGitRoot walks up from start until a .git directory or file exists (src/utils/git.ts findGitRoot).
func findGitRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	cur := filepath.Clean(abs)
	for {
		gitPath := filepath.Join(cur, ".git")
		if _, err := os.Lstat(gitPath); err == nil {
			return filepath.Clean(cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

func realPathClean(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(r)
	}
	return filepath.Clean(p)
}

// resolveCanonicalGitRoot mirrors src/utils/git.ts resolveCanonicalRoot (worktree → main repo working tree).
func resolveCanonicalGitRoot(gitRoot string) string {
	if gitRoot == "" {
		return ""
	}
	gitDot := filepath.Join(gitRoot, ".git")
	fi, err := os.Lstat(gitDot)
	if err != nil {
		return filepath.Clean(gitRoot)
	}
	if fi.IsDir() {
		return filepath.Clean(gitRoot)
	}
	data, err := os.ReadFile(gitDot)
	if err != nil {
		return filepath.Clean(gitRoot)
	}
	content := strings.TrimSpace(string(data))
	const prefix = "gitdir:"
	if !strings.HasPrefix(content, prefix) {
		return filepath.Clean(gitRoot)
	}
	rel := strings.TrimSpace(strings.TrimPrefix(content, prefix))
	worktreeGitDir := filepath.Clean(filepath.Join(gitRoot, rel))
	cdRaw, err := os.ReadFile(filepath.Join(worktreeGitDir, "commondir"))
	if err != nil {
		return filepath.Clean(gitRoot)
	}
	commonDir := filepath.Clean(filepath.Join(worktreeGitDir, strings.TrimSpace(string(cdRaw))))
	worktreesExpected := filepath.Join(commonDir, "worktrees")
	if filepath.Clean(filepath.Dir(worktreeGitDir)) != filepath.Clean(worktreesExpected) {
		return filepath.Clean(gitRoot)
	}
	gitdirBackRaw, err := os.ReadFile(filepath.Join(worktreeGitDir, "gitdir"))
	if err != nil {
		return filepath.Clean(gitRoot)
	}
	backTarget := strings.TrimSpace(string(gitdirBackRaw))
	backResolved := realPathClean(backTarget)
	gitRootResolved := realPathClean(gitRoot)
	expectedGit := filepath.Join(gitRootResolved, ".git")
	if normalizePathForComparison(backResolved) != normalizePathForComparison(expectedGit) {
		return filepath.Clean(gitRoot)
	}
	if filepath.Base(commonDir) != ".git" {
		return commonDir
	}
	return filepath.Clean(filepath.Dir(commonDir))
}

// findCanonicalGitRoot mirrors src/utils/git.ts findCanonicalGitRoot.
func findCanonicalGitRoot(startPath string) string {
	root := findGitRoot(startPath)
	if root == "" {
		return ""
	}
	return resolveCanonicalGitRoot(root)
}

// resolveStopBoundary mirrors src/utils/markdownConfigLoader.ts resolveStopBoundary.
// sessionProjectRoot is TS getProjectRoot(); if empty, use cwd so single-arg behavior matches "session = cwd".
func resolveStopBoundary(cwd, sessionProjectRoot string) string {
	sessionProj := strings.TrimSpace(sessionProjectRoot)
	if sessionProj == "" {
		sessionProj = cwd
	}
	cwdGitRoot := findGitRoot(cwd)
	sessionGitRoot := findGitRoot(sessionProj)
	if cwdGitRoot == "" || sessionGitRoot == "" {
		return cwdGitRoot
	}
	cwdCanonical := findCanonicalGitRoot(cwd)
	if cwdCanonical != "" && normalizePathForComparison(cwdCanonical) == normalizePathForComparison(sessionGitRoot) {
		return cwdGitRoot
	}
	nCwd := normalizePathForComparison(cwdGitRoot)
	nSess := normalizePathForComparison(sessionGitRoot)
	sep := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		sep = `\`
	}
	if nCwd != nSess && strings.HasPrefix(nCwd, nSess+sep) {
		return sessionGitRoot
	}
	return cwdGitRoot
}
