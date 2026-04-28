package localtools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath mirrors TS expandPath (src/utils/path.ts): expands ~ to home dir,
// resolves relative paths against baseDir, and returns a clean absolute path.
// No workspace root boundary check — access control is handled by the permission system.
func ExpandPath(filePath string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return "", fmt.Errorf("empty path")
	}

	// Handle ~ and ~/ prefix
	if trimmed == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Clean(home), nil
	}
	if strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Clean(filepath.Join(home, trimmed[2:])), nil
	}

	// Absolute paths are returned as-is (cleaned)
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	// Relative path: resolve against baseDir (or cwd if empty)
	base := baseDir
	if strings.TrimSpace(base) == "" {
		base = "."
	}
	joined := filepath.Join(base, trimmed)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", trimmed, err)
	}
	return filepath.Clean(abs), nil
}

// ResolveUnderRoots maps file_path to a clean absolute path that must lie under one of roots (prefix match).
// Relative paths are joined with primaryRoot (first element of roots, or "." if empty).
// Deprecated: use [ExpandPath] for path resolution without workspace boundary enforcement.
func ResolveUnderRoots(filePath string, roots []string) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", fmt.Errorf("empty path")
	}
	primary := "."
	if len(roots) > 0 && strings.TrimSpace(roots[0]) != "" {
		primary = roots[0]
	}
	pa, err := filepath.Abs(primary)
	if err != nil {
		return "", err
	}
	var abs string
	if filepath.IsAbs(filePath) {
		abs = filepath.Clean(filePath)
	} else {
		abs = filepath.Clean(filepath.Join(pa, filePath))
	}
	fa, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	cleanRoots := make([]string, 0, len(roots))
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		ra, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		cleanRoots = append(cleanRoots, ra)
	}
	if len(cleanRoots) == 0 {
		cleanRoots = []string{pa}
	}
	for _, root := range cleanRoots {
		if fileUnderRoot(fa, root) {
			return fa, nil
		}
	}
	return "", fmt.Errorf("path %q is outside allowed workspace roots", fa)
}

// ResolveDirUnderRoots is like [ResolveUnderRoots] but requires an existing directory.
func ResolveDirUnderRoots(dirPath string, roots []string) (string, error) {
	if strings.TrimSpace(dirPath) == "" {
		if len(roots) == 0 {
			return filepath.Abs(".")
		}
		return filepath.Abs(strings.TrimSpace(roots[0]))
	}
	abs, err := ResolveUnderRoots(dirPath, roots)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

func fileUnderRoot(fileAbs, rootAbs string) bool {
	if fileAbs == rootAbs {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(fileAbs, rootAbs+sep)
}
