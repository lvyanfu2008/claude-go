package settingsfile

import (
	"os"
	"path/filepath"
	"strings"
)

func fileExistsRegular(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// goClaudeProjectMarkers are the only project-level files Go uses to locate `.claude/` for
// env merge ([ApplyMergedClaudeSettingsEnv]) and related Go paths. Project `.claude/settings.json`
// is TypeScript-only and must not anchor Go project root discovery.
var goClaudeProjectMarkers = []string{"settings.go.json", "settings.local.json"}

// allClaudeProjectMarkers includes TS project `settings.json` for callers that read TS-shaped
// JSON (e.g. slash-resolve enabledPlugins). Prefer [FindClaudeProjectRoot] for env/settingsfile.
var allClaudeProjectMarkers = []string{"settings.json", "settings.go.json", "settings.local.json"}

func findClaudeProjectRootWithMarkers(startDir string, markers []string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	orig := dir
	for {
		cl := filepath.Join(dir, ".claude")
		for _, name := range markers {
			if fileExistsRegular(filepath.Join(cl, name)) {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return orig, nil
		}
		dir = parent
	}
}

// FindClaudeProjectRoot walks from startDir upward (including startDir) and returns
// the first directory whose `.claude/` contains **settings.go.json** or **settings.local.json**.
// Project **settings.json** is ignored on purpose (TS CLI only).
// If none is found, returns abs(startDir) so behavior matches “use cwd” when no Go marker exists.
func FindClaudeProjectRoot(startDir string) (string, error) {
	return findClaudeProjectRootWithMarkers(startDir, goClaudeProjectMarkers)
}

// FindClaudeProjectRootAny walks upward like [FindClaudeProjectRoot] but also treats
// project `.claude/settings.json` as a marker. Use for TS-compat reads (e.g. enabledPlugins);
// do not use for Go env merge — that stays on [FindClaudeProjectRoot].
func FindClaudeProjectRootAny(startDir string) (string, error) {
	return findClaudeProjectRootWithMarkers(startDir, allClaudeProjectMarkers)
}

// resolveProjectRoot returns CCB_ENGINE_PROJECT_ROOT if set, else FindClaudeProjectRoot(os.Getwd()).
func resolveProjectRoot() (string, error) {
	if r := strings.TrimSpace(os.Getenv("CCB_ENGINE_PROJECT_ROOT")); r != "" {
		return filepath.Abs(r)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindClaudeProjectRoot(wd)
}
