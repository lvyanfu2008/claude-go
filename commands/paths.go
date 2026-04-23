package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"goc/ccb-engine/debugpath"
	"goc/utils"
)

// ClaudeConfigHome returns ~/.claude or CLAUDE_CONFIG_DIR (same as TS getClaudeConfigHomeDir).
func ClaudeConfigHome() string {
	return debugpath.ConfigHomeDir()
}

// ManagedFilePath mirrors src/utils/settings/managedPath.ts getManagedFilePath.
// If CLAUDE_CODE_MANAGED_SETTINGS_PATH is set (non-empty), it is used as the base (TS: ant-only override; Go accepts for tests).
func ManagedFilePath() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH")); d != "" {
		return d
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/ClaudeCode"
	case "windows":
		return `C:\Program Files\ClaudeCode`
	default:
		return "/etc/claude-code"
	}
}

// IsEnvTruthy matches TS isEnvTruthy for common true strings.
func IsEnvTruthy(key string) bool {
	return utils.IsEnvTruthy(key)
}

// DisablePolicySkillsEnv is CLAUDE_CODE_DISABLE_POLICY_SKILLS (skips managed *skills* dir only in TS — not managed /commands).
func DisablePolicySkillsEnv() bool {
	return IsEnvTruthy("CLAUDE_CODE_DISABLE_POLICY_SKILLS")
}

// projectClaudeSubdirs matches getProjectDirsUpToHome(subdir, cwd) in src/utils/markdownConfigLoader.ts
// (uses resolveStopBoundary; most specific → ancestors; stops at home; stops at git boundary after processing it).
func projectClaudeSubdirs(cwd, subdir, sessionProjectRoot string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	home = filepath.Clean(home)
	gitStop := resolveStopBoundary(cwd, sessionProjectRoot)
	cur, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	cur = filepath.Clean(cur)
	var dirs []string
	for {
		if pathsEqual(cur, home) {
			break
		}
		sub := filepath.Join(cur, ".claude", subdir)
		if st, err := os.Stat(sub); err == nil && st.IsDir() {
			dirs = append(dirs, sub)
		}
		if gitStop != "" && normalizePathForComparison(cur) == normalizePathForComparison(gitStop) {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	dirs = appendWorktreeMainRepoProjectDirIfMissing(dirs, cwd, subdir)
	return dirs, nil
}

// appendWorktreeMainRepoProjectDirIfMissing mirrors src/utils/markdownConfigLoader.ts loadMarkdownFilesForSubdir lines 320–335
// (sparse worktree: add main repo .claude/<subdir> when worktree root lacks it).
func appendWorktreeMainRepoProjectDirIfMissing(dirs []string, cwd, subdir string) []string {
	gitRoot := findGitRoot(cwd)
	canonicalRoot := findCanonicalGitRoot(cwd)
	if gitRoot == "" || canonicalRoot == "" {
		return dirs
	}
	if normalizePathForComparison(gitRoot) == normalizePathForComparison(canonicalRoot) {
		return dirs
	}
	worktreeSubdir := filepath.Join(gitRoot, ".claude", subdir)
	nWT := normalizePathForComparison(worktreeSubdir)
	worktreeHas := false
	for _, d := range dirs {
		if normalizePathForComparison(d) == nWT {
			worktreeHas = true
			break
		}
	}
	if worktreeHas {
		return dirs
	}
	mainClaudeSubdir := filepath.Join(canonicalRoot, ".claude", subdir)
	for _, d := range dirs {
		if normalizePathForComparison(d) == normalizePathForComparison(mainClaudeSubdir) {
			return dirs
		}
	}
	return append(append([]string(nil), dirs...), mainClaudeSubdir)
}

func pathsEqual(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// IsBareMode reads env CLAUDE_CODE_BARE (truthy) — aligns with TS isBareMode for skill discovery gate.
func IsBareMode() bool {
	return IsEnvTruthy("CLAUDE_CODE_BARE")
}

// projectSkillDirs is projectClaudeSubdirs(cwd, "skills", sessionProjectRoot).
func projectSkillDirs(cwd, sessionProjectRoot string) ([]string, error) {
	return projectClaudeSubdirs(cwd, "skills", sessionProjectRoot)
}

// projectWorkflowDirs is projectClaudeSubdirs(cwd, "workflows", sessionProjectRoot) — `.claude/workflows` per ancestor.
func projectWorkflowDirs(cwd, sessionProjectRoot string) ([]string, error) {
	return projectClaudeSubdirs(cwd, "workflows", sessionProjectRoot)
}
