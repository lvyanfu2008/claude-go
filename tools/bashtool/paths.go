package bashtool

import (
	"path/filepath"
	"strings"
)

// PathSafetyResult holds the result of a path safety check.
type PathSafetyResult struct {
	Safe   bool
	Reason string
}

// ValidatePath checks whether a path is safe to access from the given working directory.
// Mirrors TS pathValidation.ts.
func ValidatePath(path, cwd string) *PathSafetyResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return &PathSafetyResult{Safe: false, Reason: "Path is empty"}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &PathSafetyResult{Safe: false, Reason: "Cannot resolve absolute path: " + err.Error()}
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return &PathSafetyResult{Safe: false, Reason: "Cannot resolve working directory: " + err.Error()}
	}

	// Check if the resolved path is within the working directory tree.
	rel, err := filepath.Rel(absCwd, absPath)
	if err != nil {
		return &PathSafetyResult{Safe: false, Reason: "Cannot compute relative path: " + err.Error()}
	}

	// Path traversal outside the working directory.
	if strings.HasPrefix(rel, "..") {
		return &PathSafetyResult{
			Safe:   false,
			Reason: "Path escapes working directory: " + rel,
		}
	}

	return &PathSafetyResult{Safe: true}
}

// dangerousPathPatterns matches paths that could access sensitive system areas.
var dangerousPathPatterns = []struct {
	pattern *string
	reason  string
}{
	{nil, ""}, // placeholder
}

// CheckDangerousPaths checks a command for references to dangerous paths
// like /etc/passwd, /etc/shadow, ~/.ssh, etc.
func CheckDangerousPaths(command string) *PathSafetyResult {
	lower := strings.ToLower(command)

	dangerousPaths := map[string]string{
		"/etc/passwd":    "references /etc/passwd",
		"/etc/shadow":    "references /etc/shadow",
		"/etc/sudoers":   "references /etc/sudoers",
		"/root/":         "references /root directory",
		"/var/log/auth":  "references auth logs",
		"/proc/":         "references /proc filesystem",
		"/sys/":          "references /sys filesystem",
		"/dev/":          "references /dev filesystem",
		"~/.ssh/":        "references SSH keys directory",
		"~/.gnupg/":      "references GPG keys directory",
		"~/.aws/":        "references AWS credentials",
		"~/.config/":     "references user config directory",
		".env":           "references .env file (may contain secrets)",
		"credentials":    "references credentials file",
		"id_rsa":         "references private SSH key",
		"id_ed25519":     "references private SSH key",
		"id_ecdsa":       "references private SSH key",
	}

	for path, reason := range dangerousPaths {
		if strings.Contains(lower, strings.ToLower(path)) {
			return &PathSafetyResult{
				Safe:   false,
				Reason: "Command " + reason,
			}
		}
	}

	return &PathSafetyResult{Safe: true}
}

// CheckPathTraversal checks for explicit path traversal attempts (../ sequences).
func CheckPathTraversal(command string) *PathSafetyResult {
	// Strip quoted content first.
	remaining := stripQuotedContent(command)

	if strings.Contains(remaining, "..") {
		return &PathSafetyResult{
			Safe:   false,
			Reason: "Command contains path traversal (..)",
		}
	}
	return &PathSafetyResult{Safe: true}
}
