package claudebase

import (
	"os"
	"path/filepath"
	"strings"
)

// ClaudeConfigHomeDir mirrors getClaudeConfigHomeDir.
func ClaudeConfigHomeDir() (string, error) {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return filepath.Clean(d), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".harness"), nil
}
