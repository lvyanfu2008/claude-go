package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// SafeResolvePath mirrors fsOperations.ts safeResolvePath (no UNC special-case beyond returning as-is).
func SafeResolvePath(filePath string) (resolvedPath string, isSymlink bool) {
	if strings.HasPrefix(filePath, `\\`) || strings.HasPrefix(filePath, "//") {
		return filePath, false
	}
	fi, err := os.Lstat(filePath)
	if err != nil {
		return filePath, false
	}
	mode := fi.Mode()
	if mode&os.ModeNamedPipe != 0 || mode&os.ModeSocket != 0 || mode.Type()&os.ModeCharDevice != 0 || mode.Type()&os.ModeDevice != 0 {
		return filePath, false
	}
	rp, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return filePath, false
	}
	return filepath.Clean(rp), rp != filepath.Clean(filePath)
}
