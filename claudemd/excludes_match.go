package claudemd

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ExcludeChecker mirrors isClaudeMdExcluded + picomatch (dot: true) via doublestar PathMatch on slash-normalized paths.
type ExcludeChecker struct {
	patterns []string
}

// NewExcludeChecker builds checker from merged settings claudeMdExcludes (after resolveExcludePatterns).
func NewExcludeChecker(patterns []string) *ExcludeChecker {
	if len(patterns) == 0 {
		return nil
	}
	return &ExcludeChecker{patterns: resolveExcludePatterns(patterns)}
}

// IsExcluded mirrors claudemd.ts isClaudeMdExcluded.
func (c *ExcludeChecker) IsExcluded(filePath string, typ MemoryType) bool {
	if c == nil || len(c.patterns) == 0 {
		return false
	}
	if typ != MemoryUser && typ != MemoryProject && typ != MemoryLocal {
		return false
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		abs = filePath
	}
	normalized := strings.ReplaceAll(filepath.ToSlash(abs), `\`, `/`)
	for _, pat := range c.patterns {
		p := strings.ReplaceAll(filepath.ToSlash(pat), `\`, `/`)
		ok, err := doublestar.Match(p, normalized)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// resolveExcludePatterns mirrors claudemd.ts resolveExcludePatterns (realpath longest static prefix of absolute patterns).
func resolveExcludePatterns(patterns []string) []string {
	expanded := make([]string, len(patterns))
	copy(expanded, patterns)
	for i, normalized := range expanded {
		expanded[i] = strings.ReplaceAll(normalized, `\`, `/`)
	}
	out := append([]string(nil), expanded...)
	for _, normalized := range expanded {
		if !strings.HasPrefix(normalized, "/") {
			continue
		}
		globStart := -1
		for idx, ch := range normalized {
			if ch == '*' || ch == '?' || ch == '{' || ch == '[' {
				globStart = idx
				break
			}
		}
		staticPrefix := normalized
		if globStart >= 0 {
			staticPrefix = normalized[:globStart]
		}
		dirToResolve := filepath.Dir(staticPrefix)
		if dirToResolve == "" || dirToResolve == "." || dirToResolve == "/" {
			continue
		}
		resolved, err := filepath.EvalSymlinks(dirToResolve)
		if err != nil || resolved == "" {
			continue
		}
		resolved = strings.ReplaceAll(filepath.ToSlash(resolved), `\`, `/`)
		prefix := strings.ReplaceAll(filepath.ToSlash(dirToResolve), `\`, `/`)
		if resolved != prefix {
			out = append(out, resolved+normalized[len(prefix):])
		}
	}
	return dedupeKeepOrder(out)
}
