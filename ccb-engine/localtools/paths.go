package localtools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveUnderRoots maps file_path to a clean absolute path that must lie under one of roots (prefix match).
// Relative paths are joined with primaryRoot (first element of roots, or "." if empty).
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
