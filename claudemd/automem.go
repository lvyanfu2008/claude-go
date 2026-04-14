package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// IsAutoMemoryEnabled mirrors src/memdir/paths.ts isAutoMemoryEnabled (env chain; settings.json omitted → default on).
func IsAutoMemoryEnabled() bool {
	envVal := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY"))
	if truthy(envVal) {
		return false
	}
	if envDefinedFalsy(envVal) {
		return true
	}
	if truthy(os.Getenv("CLAUDE_CODE_SIMPLE")) {
		return false
	}
	if truthy(os.Getenv("CLAUDE_CODE_REMOTE")) && strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")) == "" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AUTO_MEMORY_ENABLED")); v != "" {
		return truthy(v)
	}
	return true
}

// MemoryBaseDir mirrors getMemoryBaseDir.
func MemoryBaseDir() string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); v != "" {
		return filepath.Clean(v)
	}
	cfg, err := ClaudeConfigHomeDir()
	if err != nil {
		return ""
	}
	return cfg
}

// GetAutoMemPath mirrors getAutoMemPath resolution used for MEMORY.md (override env + projects/<sanitized>/memory/).
func GetAutoMemPath(originalCwd string) string {
	if p := coworkMemoryOverrideDir(); p != "" {
		return filepath.Clean(p) + string(filepath.Separator)
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")); v != "" {
		if expanded := expandTildeMemoryDir(v); expanded != "" {
			return filepath.Clean(expanded) + string(filepath.Separator)
		}
	}
	absCwd, _ := filepath.Abs(originalCwd)
	baseKey := autoMemBaseKey(absCwd)
	projectsDir := filepath.Join(MemoryBaseDir(), "projects")
	return filepath.Join(projectsDir, baseKey, "memory") + string(filepath.Separator)
}

func autoMemBaseKey(absCwd string) string {
	if cr := ResolveCanonicalGitRoot(absCwd); cr != "" {
		return SanitizePath(cr)
	}
	return SanitizePath(absCwd)
}

func coworkMemoryOverrideDir() string {
	raw := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_PATH_OVERRIDE"))
	if raw == "" {
		return ""
	}
	// Env override: absolute only, no tilde (TS validateMemoryPath expandTilde false).
	p := filepath.Clean(raw)
	if !filepath.IsAbs(p) {
		return ""
	}
	st, err := os.Stat(p)
	if err != nil {
		return ""
	}
	if st.IsDir() {
		return p
	}
	return filepath.Dir(p)
}

// EnsureMemoryDirExists mirrors ensureMemoryDirExists (memdir.ts): recursive mkdir so the model can write without checking.
func EnsureMemoryDirExists(dir string) error {
	d := strings.TrimSpace(dir)
	if d == "" {
		return nil
	}
	d = filepath.Clean(d)
	return os.MkdirAll(d, 0o700)
}

// GetTeamMemPath mirrors teamMemPaths getTeamMemPath: <autoMemPath>team/ with trailing separator.
func GetTeamMemPath(originalCwd string) string {
	auto := strings.TrimSuffix(GetAutoMemPath(originalCwd), string(filepath.Separator))
	return filepath.Join(auto, "team") + string(filepath.Separator)
}

// IsAutoMemPath mirrors src/memdir/paths.ts isAutoMemPath: absolute file path lies under
// GetAutoMemPath(originalCwd) (project-scoped auto-memory directory).
func IsAutoMemPath(absFilePath, originalCwd string) bool {
	if !IsAutoMemoryEnabled() {
		return false
	}
	p := strings.TrimSpace(absFilePath)
	if p == "" {
		return false
	}
	memRoot := strings.TrimSuffix(filepath.Clean(GetAutoMemPath(originalCwd)), string(filepath.Separator))
	if memRoot == "" || memRoot == "." {
		return false
	}
	abs := filepath.Clean(p)
	if !filepath.IsAbs(abs) {
		return false
	}
	rel, err := filepath.Rel(memRoot, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func expandTildeMemoryDir(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "~/") || strings.HasPrefix(s, `~\`) {
		rest := strings.TrimPrefix(strings.TrimPrefix(s, "~/"), `~\`)
		h, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(h, rest)
	}
	if filepath.IsAbs(s) {
		return s
	}
	return ""
}

func truthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envDefinedFalsy(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "0" || v == "false" || v == "no" || v == "off"
}
