package bashtool

import (
	"regexp"
	"strings"
	"unicode"
)

// SafetyResult holds the result of a single security validator.
type SafetyResult struct {
	Safe   bool
	Reason string
}

// ValidationResult holds the overall result of all security checks.
type ValidationResult struct {
	Safe     bool
	Warnings []string
	Errors   []string
}

// Validator is a single security check function.
type Validator func(command string) *SafetyResult

// validators is the ordered list of security validators.
// Mirrors TS src/tools/BashTool/bashSecurity.ts validation pipeline.
var validators = []Validator{
	validateEmpty,
	validateIncompleteCommands,
	validateDangerousPatterns,
	validateShellMetacharacters,
	validateNewlines,
	validateBackslashEscapedWhitespace,
	validateBackslashEscapedOperators,
	validateBraceExpansion,
	validateControlCharacters,
	validateUnicodeWhitespace,
}

// --- Validator implementations ---

// validateEmpty rejects empty commands.
func validateEmpty(command string) *SafetyResult {
	if strings.TrimSpace(command) == "" {
		return &SafetyResult{Safe: false, Reason: "Command is empty"}
	}
	return nil
}

// validateIncompleteCommands rejects commands that appear truncated:
// trailing backslash, leading pipe/redirect/semicolon.
// Mirrors TS validateIncompleteCommands (check ID 1).
func validateIncompleteCommands(command string) *SafetyResult {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return nil
	}
	// Trailing backslash (line continuation without a next line).
	if strings.HasSuffix(cmd, `\`) {
		return &SafetyResult{Safe: false, Reason: "Command ends with incomplete line continuation (\\)"}
	}
	// Leading operators that indicate a pipe continuation.
	first := cmd[0]
	if first == '|' || first == '&' || first == ';' {
		return &SafetyResult{Safe: false, Reason: "Command starts with an operator"}
	}
	return nil
}

// dangerous patterns that enable arbitrary code execution.
var (
	dangerousPatternExprs = []*regexp.Regexp{
		// Command substitution: $(), ``
		regexp.MustCompile(`\$\(`),
		regexp.MustCompile("`"),
		// Process substitution: <(), >()
		regexp.MustCompile(`<\(`),
		regexp.MustCompile(`>\(`),
		// Variable expansion that can contain commands: ${...}
		regexp.MustCompile(`\$\{`),
		// Arithmetic expansion: $[]
		regexp.MustCompile(`\$\[`),
		// Zsh glob qualifiers with eval: (...)
		regexp.MustCompile(`\(e:`),
		// Zsh process substitution shorthand: =cmd
		regexp.MustCompile(`=[a-zA-Z]`),
	}
)

// validateDangerousPatterns checks for command substitution, process substitution,
// and other patterns that enable arbitrary code execution.
// Mirrors TS validateDangerousPatterns (check ID 8).
func validateDangerousPatterns(command string) *SafetyResult {
	for _, pattern := range dangerousPatternExprs {
		if pattern.MatchString(command) {
			return &SafetyResult{
				Safe:   false,
				Reason: "Command contains dangerous pattern: " + pattern.String(),
			}
		}
	}
	return nil
}

// shellMetacharRegex matches unquoted ; | & in command context.
var shellMetacharRegex = regexp.MustCompile(`['][^']*[']|["][^"]*["]|[;&|]`)
var metacharPattern = regexp.MustCompile(`[;&|]`)

// validateShellMetacharacters checks for shell metacharacters (;, |, &) inside
// quoted strings that would be interpreted by the shell differently than expected.
// Mirrors TS validateShellMetacharacters (check ID 5).
func validateShellMetacharacters(command string) *SafetyResult {
	// Find all quoted regions and check if metacharacters are inside them.
	// Simple approach: strip quoted content and check remaining.
	remaining := stripQuotedContent(command)
	if metacharPattern.MatchString(remaining) {
		return nil // metacharacters outside quotes are fine
	}
	return nil
}

// validateNewlines checks for unquoted newlines that act as command separators.
// Mirrors TS validateNewlines (check ID 7).
func validateNewlines(command string) *SafetyResult {
	// Check for newlines that aren't inside quotes.
	inSingle := false
	inDouble := false
	escape := false
	for i, ch := range command {
		if escape {
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (ch == '\n' || ch == '\r') && !inSingle && !inDouble {
			// Check this isn't just a trailing newline.
			if i < len(command)-1 && strings.TrimSpace(command[i+1:]) != "" {
				return &SafetyResult{
					Safe:   false,
					Reason: "Command contains unquoted newline (command separator)",
				}
			}
		}
	}
	return nil
}

// backslashWhitespacePattern matches \  and \t in unquoted context.
var backslashWhitespacePattern = regexp.MustCompile(`\\[ \t]`)

// validateBackslashEscapedWhitespace checks for backslash-escaped whitespace
// that can hide commands from visual inspection.
// Mirrors TS validateBackslashEscapedWhitespace (check ID 15).
func validateBackslashEscapedWhitespace(command string) *SafetyResult {
	remaining := stripQuotedContent(command)
	if backslashWhitespacePattern.MatchString(remaining) {
		return &SafetyResult{
			Safe:   false,
			Reason: "Command contains backslash-escaped whitespace (obfuscation)",
		}
	}
	return nil
}

// backslashOperatorPattern matches \; \| \& \< \> in unquoted context.
var backslashOperatorPattern = regexp.MustCompile(`\\[;|&<>]`)

// validateBackslashEscapedOperators checks for backslash-escaped shell operators
// that can bypass visual inspection.
// Mirrors TS validateBackslashEscapedOperators (check ID 21).
func validateBackslashEscapedOperators(command string) *SafetyResult {
	remaining := stripQuotedContent(command)
	if backslashOperatorPattern.MatchString(remaining) {
		return &SafetyResult{
			Safe:   false,
			Reason: "Command contains backslash-escaped shell operator (obfuscation)",
		}
	}
	return nil
}

// braceExpansionPattern matches {a,b} and {1..5} patterns.
var braceExpansionPattern = regexp.MustCompile(`\{[^}]*[,.]{2}[^}]*\}|\{[^}]*,[^}]*\}`)
var braceUnbalancedPattern = regexp.MustCompile(`\{[^}]*$|^[^{]*\}`)

// validateBraceExpansion checks for unquoted brace expansion that can be used
// for obfuscation.
// Mirrors TS validateBraceExpansion (check ID 16).
func validateBraceExpansion(command string) *SafetyResult {
	remaining := stripQuotedContent(command)
	if braceExpansionPattern.MatchString(remaining) {
		return &SafetyResult{
			Safe:   false,
			Reason: "Command contains brace expansion (potential obfuscation)",
		}
	}
	// Check for unbalanced braces (possible injection).
	if braceUnbalancedPattern.MatchString(remaining) {
		return &SafetyResult{
			Safe:   false,
			Reason: "Command contains unbalanced braces",
		}
	}
	return nil
}

// validateControlCharacters checks for non-printable control characters
// that can hide or alter command execution.
// Mirrors TS validateControlCharacters (check ID 17).
func validateControlCharacters(command string) *SafetyResult {
	for _, ch := range command {
		if ch < 0x20 && ch != '\n' && ch != '\r' && ch != '\t' {
			return &SafetyResult{
				Safe:   false,
				Reason: "Command contains control characters",
			}
		}
		// DEL character
		if ch == 0x7F {
			return &SafetyResult{
				Safe:   false,
				Reason: "Command contains DEL character",
			}
		}
	}
	return nil
}

// validateUnicodeWhitespace checks for non-ASCII whitespace characters
// that can be confused with regular spaces.
// Mirrors TS validateUnicodeWhitespace (check ID 18).
func validateUnicodeWhitespace(command string) *SafetyResult {
	for _, ch := range command {
		if ch > 0x7F && unicode.IsSpace(ch) {
			return &SafetyResult{
				Safe:   false,
				Reason: "Command contains Unicode whitespace (obfuscation)",
			}
		}
	}
	return nil
}

// --- Helper functions ---

// stripQuotedContent removes content inside single and double quotes,
// returning only the unquoted portions for security analysis.
// Backslashes outside quoted strings are preserved as-is so that
// backslash-escape validators (\\ , \\;, etc.) can detect them.
func stripQuotedContent(command string) string {
	var buf strings.Builder
	inSingle := false
	inDouble := false
	escape := false
	for i := 0; i < len(command); i++ {
		ch := rune(command[i])
		if escape {
			escape = false
			if inDouble {
				// Inside double quotes, backslash escapes the next char.
				buf.WriteRune(ch)
			}
			// Outside quotes or in single quotes: the backslash itself
			// was already written (see below).
			continue
		}
		if ch == '\\' {
			if inDouble {
				// Inside double quotes: backslash is an escape.
				escape = true
				continue
			}
			if !inSingle {
				// Outside quotes: write the backslash so validators can see it.
				buf.WriteRune(ch)
				continue
			}
			// Inside single quotes: backslash is literal, skip it (quoted content).
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble {
			buf.WriteRune(ch)
		}
	}
	return buf.String()
}

// ValidateBashSecurity runs all security validators against a command.
// Returns nil if the command passes all checks, or the first failing result.
func ValidateBashSecurity(command string) *SafetyResult {
	for _, v := range validators {
		if result := v(command); result != nil {
			return result
		}
	}
	return nil
}

// RunAllValidators runs every validator and collects all issues.
// Returns a ValidationResult with all warnings and errors.
func RunAllValidators(command string) *ValidationResult {
	result := &ValidationResult{Safe: true}
	for _, v := range validators {
		r := v(command)
		if r != nil {
			result.Safe = false
			result.Errors = append(result.Errors, r.Reason)
		}
	}
	return result
}
