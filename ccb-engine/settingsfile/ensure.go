package settingsfile

import (
	"sync"
)

var (
	ensureOnce            sync.Once
	ensureErr             error
	lastResolvedProjectMu sync.RWMutex
	lastResolvedProject   string
)

// ProjectRootLastResolved returns the project directory used for .claude/settings*.json
// after the first successful EnsureProjectClaudeEnvOnce (empty before that).
func ProjectRootLastResolved() string {
	lastResolvedProjectMu.RLock()
	defer lastResolvedProjectMu.RUnlock()
	return lastResolvedProject
}

// EnsureProjectClaudeEnvOnce loads merged settings env (see ApplyMergedClaudeSettingsEnv).
// Project root: $CCB_ENGINE_PROJECT_ROOT if set; otherwise the nearest ancestor of
// the current working directory whose .claude/ contains **settings.go.json** or
// **settings.local.json** (project settings.json is TS-only and is not a marker); if none,
// the starting directory’s abs path (so nested `goc/` runs without a Go marker still use cwd).
func EnsureProjectClaudeEnvOnce() error {
	ensureOnce.Do(func() {
		root, err := resolveProjectRoot()
		if err != nil {
			ensureErr = err
			return
		}
		lastResolvedProjectMu.Lock()
		lastResolvedProject = root
		lastResolvedProjectMu.Unlock()
		ensureErr = ApplyMergedClaudeSettingsEnv(root)
	})
	return ensureErr
}

// ResetEnsureForTesting clears the once-state (unit tests only).
func ResetEnsureForTesting() {
	ensureOnce = sync.Once{}
	ensureErr = nil
	lastResolvedProjectMu.Lock()
	lastResolvedProject = ""
	lastResolvedProjectMu.Unlock()
}
