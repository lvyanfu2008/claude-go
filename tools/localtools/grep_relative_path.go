package localtools

import (
	"path/filepath"
	"strings"
)

// ToRelativePathFromCWD mirrors toRelativePath in src/utils/path.ts (cwd = TS getCwd()).
// If the relative path would start with "..", returns absPath unchanged.
func ToRelativePathFromCWD(cwd, absPath string) string {
	cwd = filepath.Clean(cwd)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	if rel == "." {
		return "."
	}
	if strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(absPath)
	}
	return filepath.ToSlash(rel)
}
