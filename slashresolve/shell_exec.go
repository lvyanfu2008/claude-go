package slashresolve

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ExecuteShellCommandsInPrompt finds shell command substitution patterns in prompt
// text and replaces them with command output. Supports two patterns:
//   - ${...} — inline shell execution, substituted inline
//   - `backtick commands` — backtick execution, substituted inline
//
// TS parity: src/commands/skill/executeShellCommandsInPrompt.ts
// Safety: commands are executed via sh -c. No sandboxing is applied (matches TS behavior).
func ExecuteShellCommandsInPrompt(prompt string) (string, error) {
	var errs []string
	result := prompt

	// Replace ${...} patterns: ${command}
	dollarBraceRe := regexp.MustCompile(`\$\{([^}]+)\}`)
	result = dollarBraceRe.ReplaceAllStringFunc(result, func(match string) string {
		cmd := strings.TrimSpace(match[2 : len(match)-1]) // strip ${ and }
		if cmd == "" {
			return match
		}
		out, err := runShellCommand(cmd)
		if err != nil {
			errs = append(errs, fmt.Sprintf("${%s}: %v", cmd, err))
			return match // leave original on error
		}
		return strings.TrimRight(out, "\n\r")
	})

	// Replace backtick patterns: `command`
	// Simple approach: find paired backticks that are NOT inside ${...}.
	backtickRe := regexp.MustCompile("`([^`]+)`")
	result = backtickRe.ReplaceAllStringFunc(result, func(match string) string {
		cmd := strings.TrimSpace(match[1 : len(match)-1]) // strip backticks
		if cmd == "" {
			return match
		}
		// Skip if this backtick is inside a ${...} that wasn't already replaced.
		// Since dollar-brace replacement runs first, remaining backticks are safe.
		out, err := runShellCommand(cmd)
		if err != nil {
			errs = append(errs, fmt.Sprintf("`%s`: %v", cmd, err))
			return match // leave original on error
		}
		return strings.TrimRight(out, "\n\r")
	})

	if len(errs) > 0 {
		return result, fmt.Errorf("shell exec errors: %s", strings.Join(errs, "; "))
	}
	return result, nil
}

func runShellCommand(cmd string) (string, error) {
	c := exec.Command("sh", "-c", cmd)
	out, err := c.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("exit %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
