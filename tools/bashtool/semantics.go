package bashtool

import "strings"

// CommandSemantic interprets a command's exit code to determine if it's an error.
// Some commands (grep, diff, find, test) use non-zero exit codes to signal
// conditional success rather than failure.
// Mirrors TS src/tools/BashTool/commandSemantics.ts
type CommandSemantic struct {
	ExitCode1    string // message when exit code is 1 (not an error)
	ErrorMin     int    // exit codes >= this are errors
}

// InterpretResult contains the interpretation of a command's exit.
type InterpretResult struct {
	IsError bool
	Message string
}

var commandSemantics = map[string]CommandSemantic{
	"grep": {ExitCode1: "No matches found", ErrorMin: 2},
	"rg":   {ExitCode1: "No matches found", ErrorMin: 2},
	"find": {ExitCode1: "Some directories were inaccessible", ErrorMin: 2},
	"diff": {ExitCode1: "Files differ", ErrorMin: 2},
	"test": {ExitCode1: "Condition is false", ErrorMin: 2},
	"[":    {ExitCode1: "Condition is false", ErrorMin: 2},
}

// InterpretCommandResult interprets a command's exit code using its semantics.
// Returns an InterpretResult indicating whether the exit code should be treated
// as an error and an optional human-readable message.
func InterpretCommandResult(command string, exitCode int) InterpretResult {
	if exitCode == 0 {
		return InterpretResult{IsError: false}
	}

	base := extractBaseCommand(command)
	if cs, ok := commandSemantics[base]; ok {
		if exitCode == 1 {
			return InterpretResult{IsError: false, Message: cs.ExitCode1}
		}
		if exitCode >= cs.ErrorMin {
			return InterpretResult{IsError: true}
		}
		return InterpretResult{IsError: false}
	}

	// Default: any non-zero exit code is an error.
	return InterpretResult{IsError: exitCode != 0}
}

// extractBaseCommand returns the first word of a command, skipping leading
// whitespace.
func extractBaseCommand(command string) string {
	cmd := strings.TrimSpace(command)
	// Handle piped commands: use the last command's semantics.
	if idx := strings.LastIndex(cmd, "|"); idx >= 0 {
		cmd = strings.TrimSpace(cmd[idx+1:])
	}
	// Get the first whitespace-delimited token.
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
