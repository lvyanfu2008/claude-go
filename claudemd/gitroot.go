package claudemd

import (
	"path/filepath"
	"strings"
)

// FindGitRoot and ResolveCanonicalGitRoot have moved to goc/claudebase.

// PathInWorkingPath approximates src/utils/permissions/filesystem.ts pathInWorkingPath for nested-worktree skip.
func PathInWorkingPath(path, workingPath string) bool {
	ap, err1 := filepath.Abs(path)
	aw, err2 := filepath.Abs(workingPath)
	if err1 != nil || err2 != nil {
		return false
	}
	ap = normalizePrivateTmp(ap)
	aw = normalizePrivateTmp(aw)
	rel, err := filepath.Rel(aw, ap)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func normalizePrivateTmp(p string) string {
	// macOS: /private/tmp -> /tmp style normalization for comparison
	if strings.HasPrefix(p, "/private/tmp/") {
		return "/tmp/" + strings.TrimPrefix(p, "/private/tmp/")
	}
	if p == "/private/tmp" {
		return "/tmp"
	}
	if strings.HasPrefix(p, "/private/var/") {
		return "/var/" + strings.TrimPrefix(p, "/private/var/")
	}
	return p
}
