package bashtool

import (
	"regexp"
	"strings"
)

// ReadOnlyResult holds the result of a readonly check.
type ReadOnlyResult struct {
	ReadOnly bool
	Reason   string
}

// readonlyCommandRegexes holds patterns for commands that are always safe
// when used without shell metacharacters, redirects, or variable expansion.
// Mirrors TS READONLY_COMMANDS + READONLY_COMMAND_REGEXES.
var readonlyCommandRegexes = []*regexp.Regexp{
	// Simple commands validated via makeRegexForSafeCommand.
	regexp.MustCompile(`^(?:echo|pwd|whoami|true|false|sleep|which|type|id|uname|free|df|du|locale|groups|nproc|basename|dirname|realpath|readlink|uptime|arch|hostname|stat|wc|wc|strings|hexdump|od|nl|cut|paste|tr|column|tac|rev|fold|expand|unexpand|fmt|comm|cmp|numfmt|seq|tsort|pr|expr|test|getconf|cal)(?:\s|$)`),

	// cat/head/tail — file viewing
	regexp.MustCompile(`^(?:cat|head|tail)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// ls — directory listing (block redirects and dangerous flags)
	regexp.MustCompile(`^ls(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// cd — change directory (no shell metacharacters)
	regexp.MustCompile(`^cd(?:\s+(?:'[^']*'|"[^"]*"|[^\s;|&` + "`" + `$(){}><#\\]+))?$`),

	// find — dangerous flags (-exec, -delete, etc.) are caught by dangerousReadonlyFlags
	regexp.MustCompile(`^find(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// grep — readonly flags only, block -f (file input could be dangerous)
	regexp.MustCompile(`^(?:grep|rg)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// diff
	regexp.MustCompile(`^diff(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// sort/uniq
	regexp.MustCompile(`^(?:sort|uniq)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// file
	regexp.MustCompile(`^file(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// jq — readonly (block -f/--from-file/--rawfile/--slurpfile/--run-tests/-L/--library-path/env/$ENV)
	regexp.MustCompile(`^jq(?:\s+(?:-[a-zA-Z]+|--[a-zA-Z-]+(?:=\S+)?))*(?:\s+'[^'` + "`" + `]*'|\s+"[^"` + "`" + `]*"|\s+[^-\s'"][^\s]*)*\s*$`),

	// man/help/info
	regexp.MustCompile(`^(?:man|help|info)(?:\s|$)`),

	// docker — readonly subcommands (ps, images, logs, inspect)
	regexp.MustCompile(`^docker\s+(?:ps|images|logs|inspect)(?:\s|$)`),

	// git readonly subcommands
	regexp.MustCompile(`^git\s+(?:diff|log|show|status|blame|branch|tag|stash\s+list|ls-files|ls-remote|ls-tree|rev-parse|rev-list|describe|cat-file|for-each-ref|merge-base|remote|remote\s+show|worktree\s+list|config\s+--get|shortlog|grep|stash\s+show|reflog)(?:\s|$)`),

	// node/python version checks
	regexp.MustCompile(`^(?:node|python|python3)\s+(?:-v|--version)$`),

	// history
	regexp.MustCompile(`^history(?:\s+\d+)?\s*$`),

	// alias
	regexp.MustCompile(`^alias$`),

	// ip/ifconfig
	regexp.MustCompile(`^(?:ip\s+addr|ifconfig(?:\s+[a-zA-Z][a-zA-Z0-9_-]*)?)\s*$`),

	// netstat/ss
	regexp.MustCompile(`^(?:netstat|ss)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// ps/pgrep
	regexp.MustCompile(`^(?:ps|pgrep)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// tree
	regexp.MustCompile(`^tree(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// date/hostname
	regexp.MustCompile(`^(?:date|hostname)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// base64/sha256sum/sha1sum/md5sum
	regexp.MustCompile(`^(?:base64|sha256sum|sha1sum|md5sum|shasum)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),

	// fd/fdfind
	regexp.MustCompile(`^(?:fd|fdfind)(?:\s+[^<>()$` + "`" + `|{}&;\n\r]*)?$`),
}

// dangerousReadonlyFlags are flags that turn a normally-readonly command dangerous.
var dangerousReadonlyFlags = []*regexp.Regexp{
	// find: code execution / file deletion flags
	// Use (?:^|\s) and (?:\s|$) instead of \b because \b doesn't match
	// between a space and a hyphen (both non-word characters).
	regexp.MustCompile(`(?:^|\s)-(?:exec|execdir|ok|okdir|delete)(?:\s|$)`),
	// git: destructive operations
	regexp.MustCompile(`\bgit\s+reset\s+--hard\b`),
	regexp.MustCompile(`\bgit\s+push\s+--force\b`),
	regexp.MustCompile(`\bgit\s+clean\s+-f\b`),
	regexp.MustCompile(`\bgit\s+branch\s+-D\b`),
}

// isCommandReadOnly checks if a command is in the readonly allowlist.
// Mirrors TS isCommandReadOnly / isReadOnlyCommand.
func isCommandReadOnly(command string) *ReadOnlyResult {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return &ReadOnlyResult{ReadOnly: false, Reason: "Command is empty"}
	}

	// First check: does any readonly regex match?
	for _, re := range readonlyCommandRegexes {
		if re.MatchString(cmd) {
			// Second check: no dangerous flags embedded
			for _, dangerous := range dangerousReadonlyFlags {
				if dangerous.MatchString(cmd) {
					return &ReadOnlyResult{
						ReadOnly: false,
						Reason:   "Command contains dangerous flags: " + dangerous.String(),
					}
				}
			}
			return &ReadOnlyResult{ReadOnly: true}
		}
	}

	return &ReadOnlyResult{ReadOnly: false, Reason: "Command is not in the readonly allowlist"}
}

// CheckReadOnlyConstraints runs the full readonly validation pipeline.
// Mirrors TS checkReadOnlyConstraints.
func CheckReadOnlyConstraints(command string) *ReadOnlyResult {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return &ReadOnlyResult{ReadOnly: false, Reason: "Command is empty"}
	}

	// Split compound commands (;) and check each subcommand.
	subcommands := splitCompoundCommands(cmd)
	for i, sub := range subcommands {
		sub = strings.TrimSpace(sub)
		if sub == "" {
			continue
		}

		// Skip variable assignments (VAR=value command)
		if isVariableAssignment(sub) {
			continue
		}

		result := isCommandReadOnly(sub)
		if !result.ReadOnly {
			if len(subcommands) > 1 {
				result.Reason = "Subcommand " + string(rune('0'+i+1)) + " is not readonly: " + result.Reason
			}
			return result
		}
	}

	return &ReadOnlyResult{ReadOnly: true}
}

// splitCompoundCommands splits a command by top-level ; separators.
func splitCompoundCommands(command string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escape := false

	for _, ch := range command {
		if escape {
			current.WriteRune(ch)
			escape = false
			continue
		}
		if ch == '\\' {
			current.WriteRune(ch)
			escape = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteRune(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteRune(ch)
			continue
		}
		if ch == ';' && !inSingle && !inDouble {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// isVariableAssignment checks if a command starts with a variable assignment (VAR=value).
func isVariableAssignment(command string) bool {
	cmd := strings.TrimSpace(command)
	eqIdx := strings.Index(cmd, "=")
	if eqIdx <= 0 {
		return false
	}
	name := cmd[:eqIdx]
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}
