// Dynamic skill directory discovery (TS discoverSkillDirsForPaths + git check-ignore).
package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverSkillDirsForPaths mirrors src/skills/loadSkillsDir.ts discoverSkillDirsForPaths.
// It walks up from each file's directory toward cwd (exclusive of cwd) and collects
// existing <dir>/.claude/skills paths where <dir> is not gitignored (git check-ignore).
//
// persistentSeen: if non-nil, skillDir paths are recorded here and skipped on later calls
// (TS process-global dynamicSkillDirs). If nil, a new map is used for dedup within this call only.
//
// Sort order: deepest paths first (same as TS split(pathSep).length descending).
func DiscoverSkillDirsForPaths(filePaths []string, cwd string, persistentSeen map[string]struct{}) []string {
	cwd = filepath.Clean(strings.TrimSpace(cwd))
	if cwd == "" || len(filePaths) == 0 {
		return nil
	}
	seen := persistentSeen
	if seen == nil {
		seen = make(map[string]struct{})
	}
	var newDirs []string

	for _, fp := range filePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		fp = filepath.Clean(fp)
		currentDir := filepath.Dir(fp)
		for isStrictChildOf(currentDir, cwd) {
			skillDir := filepath.Join(currentDir, ".claude", "skills")
			if _, done := seen[skillDir]; !done {
				seen[skillDir] = struct{}{}
				st, err := os.Stat(skillDir)
				if err == nil && st.IsDir() && !IsPathGitignored(currentDir, cwd) {
					newDirs = append(newDirs, skillDir)
				}
			}
			next := parentDir(currentDir)
			if next == currentDir {
				break
			}
			currentDir = next
		}
	}

	sort.SliceStable(newDirs, func(i, j int) bool {
		di := pathDepthScore(newDirs[i])
		dj := pathDepthScore(newDirs[j])
		if di != dj {
			return di > dj
		}
		return newDirs[i] > newDirs[j]
	})
	return newDirs
}

func isStrictChildOf(dir, root string) bool {
	d := filepath.Clean(dir)
	r := filepath.Clean(root)
	if d == r {
		return false
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(d+sep, r+sep)
}

func parentDir(dir string) string {
	d := filepath.Dir(dir)
	if d == dir {
		return dir
	}
	return d
}

func pathDepthScore(p string) int {
	s := filepath.ToSlash(filepath.Clean(p))
	if s == "" || s == "." {
		return 0
	}
	return strings.Count(s, "/")
}
