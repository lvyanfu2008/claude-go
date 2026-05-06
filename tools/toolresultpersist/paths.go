package toolresultpersist

import (
	"os"
	"path/filepath"
	"strings"

	"goc/memdir"
)

// SessionInfo carries the minimal session identity needed to compute tool-result paths.
type SessionInfo struct {
	SessionID string // session UUID
	Cwd       string // original cwd for resolving the project directory
}

// GetToolResultsDir returns the session's tool-results directory path:
//
//	{configHome}/projects/{sanitized_path}/{sessionId}/tool-results/
func GetToolResultsDir(info SessionInfo) string {
	base := memdir.ClaudeProjectSessionDir(info.Cwd)
	if base == "" {
		return ""
	}
	return filepath.Join(base, info.SessionID, ToolResultsSubdir) + string(filepath.Separator)
}

// GetToolResultPath returns the full file path for a persisted tool result.
// Extension is "json" for JSON array content, "txt" for string content.
func GetToolResultPath(info SessionInfo, toolUseID string, isJSON bool) string {
	ext := "txt"
	if isJSON {
		ext = "json"
	}
	return filepath.Join(GetToolResultsDir(info), toolUseID+"."+ext)
}

// EnsureToolResultsDir creates the session-specific tool results directory
// (recursive, idempotent). Returns the directory path or an error.
func EnsureToolResultsDir(info SessionInfo) (string, error) {
	dir := strings.TrimSuffix(GetToolResultsDir(info), string(filepath.Separator))
	if dir == "" {
		return "", nil
	}
	return dir, os.MkdirAll(dir, 0o700)
}
