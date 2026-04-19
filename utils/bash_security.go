package utils

import (
	"strings"
)

// BashSecurityCheckResult represents the result of a security check
type BashSecurityCheckResult struct {
	Allowed  bool
	Reason   string
	Ask      bool // Whether to ask for permission
	Deny     bool // Whether to deny the command
}

// CheckBashCommandSecurity checks a bash command for security issues
// Simplified version of TS security checks focusing on Git-related issues
func CheckBashCommandSecurity(command string) BashSecurityCheckResult {
	// Trim and check empty
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return BashSecurityCheckResult{Allowed: false, Reason: "Empty command"}
	}

	// Split by pipe to check segments
	segments := splitByPipes(cmd)

	// Check for multiple cd commands across segments
	cdCount := 0
	for _, segment := range segments {
		subcommands := splitCompoundCommands(segment)
		for _, subcmd := range subcommands {
			trimmed := strings.TrimSpace(subcmd)
			if IsNormalizedCdCommand(trimmed) {
				cdCount++
			}
		}
	}
	if cdCount > 1 {
		return BashSecurityCheckResult{
			Allowed: false,
			Ask:     true,
			Reason:  "Multiple directory changes in one command require approval for clarity",
		}
	}

	// Check for cd+git across pipe segments (security check)
	if CheckCdGitCompoundCommand(segments) {
		return BashSecurityCheckResult{
			Allowed: false,
			Ask:     true,
			Reason:  "Compound commands with cd and git require approval to prevent bare repository attacks",
		}
	}

	// Check each segment for individual security issues
	for _, segment := range segments {
		result := checkSegmentSecurity(segment)
		if !result.Allowed {
			return result
		}
	}

	return BashSecurityCheckResult{Allowed: true}
}

// checkSegmentSecurity checks a single command segment
func checkSegmentSecurity(segment string) BashSecurityCheckResult {
	// Split compound commands by &&
	subcommands := splitCompoundCommands(segment)

	// Check each subcommand
	for _, subcmd := range subcommands {
		trimmed := strings.TrimSpace(subcmd)

		// Skip empty subcommands
		if trimmed == "" {
			continue
		}

		// Check for dangerous patterns
		if isDangerousPattern(trimmed) {
			return BashSecurityCheckResult{
				Allowed: false,
				Deny:    true,
				Reason:  "Command contains dangerous pattern: " + trimmed,
			}
		}

		// Check for write operations in protected directories
		if isWriteToProtectedDirectory(trimmed) {
			return BashSecurityCheckResult{
				Allowed: false,
				Ask:     true,
				Reason:  "Write operation to protected directory requires approval",
			}
		}
	}

	return BashSecurityCheckResult{Allowed: true}
}

// splitByPipes splits command by pipe characters, handling quoted pipes
func splitByPipes(command string) []string {
	var segments []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escapeNext := false

	for _, ch := range command {
		if escapeNext {
			current.WriteRune(ch)
			escapeNext = false
			continue
		}

		switch ch {
		case '\\':
			escapeNext = true
			current.WriteRune(ch)
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			current.WriteRune(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(ch)
		case '|':
			if !inSingleQuote && !inDoubleQuote {
				segments = append(segments, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// splitCompoundCommands splits by && and ;, handling quoted operators
func splitCompoundCommands(segment string) []string {
	var subcommands []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escapeNext := false

	runes := []rune(segment)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if escapeNext {
			current.WriteRune(ch)
			escapeNext = false
			continue
		}

		switch ch {
		case '\\':
			escapeNext = true
			current.WriteRune(ch)
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			current.WriteRune(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(ch)
		case '&':
			// Check for &&
			if i+1 < len(runes) && runes[i+1] == '&' && !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					subcommands = append(subcommands, current.String())
					current.Reset()
				}
				i++ // Skip next &
			} else {
				current.WriteRune(ch)
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					subcommands = append(subcommands, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		subcommands = append(subcommands, current.String())
	}

	return subcommands
}

// isDangerousPattern checks for known dangerous command patterns
func isDangerousPattern(command string) bool {
	// Simple dangerous pattern detection
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		":(){ :|:& };:", // Fork bomb
		"mkfs",          // Format commands
		"dd if=/dev/",
		"> /dev/sda",
	}

	lowerCmd := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return true
		}
	}

	return false
}

// isWriteToProtectedDirectory checks for writes to protected directories
func isWriteToProtectedDirectory(command string) bool {
	// Simple check for writes to system directories
	protectedDirs := []string{
		"/bin/",
		"/sbin/",
		"/usr/bin/",
		"/usr/sbin/",
		"/etc/",
		"/boot/",
		"/lib/",
		"/lib64/",
		"/root/",
		"/var/log/",
	}

	// Check for redirection to protected directories
	for _, dir := range protectedDirs {
		if strings.Contains(command, "> "+dir) || strings.Contains(command, ">> "+dir) {
			return true
		}
	}

	// Check for commands that write to protected directories
	writeCommands := []string{
		"cp ",
		"mv ",
		"install ",
		"touch ",
	}

	for _, wcmd := range writeCommands {
		if strings.HasPrefix(command, wcmd) {
			// Extract target path
			parts := strings.Fields(command)
			if len(parts) > 1 {
				target := parts[len(parts)-1]
				for _, dir := range protectedDirs {
					if strings.HasPrefix(target, dir) {
						return true
					}
				}
			}
		}
	}

	return false
}

// SplitCommandSimple splits a command into subcommands by && and ;
// Simplified version for permission checking
func SplitCommandSimple(command string) []string {
	return splitCompoundCommands(command)
}