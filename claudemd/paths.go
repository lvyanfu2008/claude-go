package claudemd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"goc/claudebase"
)

// ClaudeConfigHomeDir has moved to goc/claudebase.

// ManagedFilePath mirrors settings/managedPath.ts getManagedFilePath.
func ManagedFilePath() string {
	if strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" {
		if p := strings.TrimSpace(os.Getenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH")); p != "" {
			return filepath.Clean(p)
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/ClaudeCode"
	case "windows":
		return `C:\Program Files\ClaudeCode`
	default:
		return "/etc/claude-code"
	}
}

// MemoryPath mirrors config.ts getMemoryPath (subset used by getMemoryFiles).
func MemoryPath(memoryType MemoryType, _ string) string {
	cfg, err := claudebase.ClaudeConfigHomeDir()
	if err != nil {
		cfg = ""
	}
	switch memoryType {
	case MemoryUser:
		return filepath.Join(cfg, "CLAUDE.md")
	case MemoryManaged:
		return filepath.Join(ManagedFilePath(), "CLAUDE.md")
	default:
		return ""
	}
}

func managedClaudeRulesDir() string {
	return filepath.Join(ManagedFilePath(), ".claude", "rules")
}

func userClaudeRulesDir() (string, error) {
	cfg, err := claudebase.ClaudeConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "rules"), nil
}
