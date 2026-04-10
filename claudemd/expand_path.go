package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath mirrors src/utils/path.ts expandPath (Unix + absolute; Windows drive paths via filepath).
func ExpandPath(path, baseDir string) string {
	if strings.Contains(path, "\x00") || strings.Contains(baseDir, "\x00") {
		b, _ := filepath.Abs(baseDir)
		return b
	}
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		b, _ := filepath.Abs(baseDir)
		return filepath.Clean(b)
	}
	if trimmed == "~" {
		h, _ := os.UserHomeDir()
		return h
	}
	if strings.HasPrefix(trimmed, "~/") || (len(trimmed) > 2 && trimmed[0] == '~' && (trimmed[1] == '/' || trimmed[1] == '\\')) {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, trimmed[2:])
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Join(baseDir, trimmed)
}
