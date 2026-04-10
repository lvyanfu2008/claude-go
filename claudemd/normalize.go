package claudemd

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePathForComparison mirrors src/utils/file.ts normalizePathForComparison.
func NormalizePathForComparison(filePath string) string {
	normalized := filepath.Clean(filePath)
	if runtime.GOOS == "windows" {
		normalized = strings.ReplaceAll(normalized, "/", `\`)
		normalized = strings.ToLower(normalized)
	}
	return normalized
}
