package slashresolve

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ExecuteShellCommandsInPrompt finds shell command substitution patterns in prompt
// text and replaces them with command output. Supports two patterns:
//   - ```! command ``` — block shell execution (replaced inline with output)
//   - !`command` — inline shell execution (must be preceded by ^ or whitespace)
//
// TS parity: src/utils/promptShellExecution.ts
// Safety: commands are executed via sh -c. No sandboxing is applied (matches TS behavior).
func ExecuteShellCommandsInPrompt(prompt string) (string, error) {
	var errs []string
	result := prompt

	// Block pattern: ```! command ``` (matches TS BLOCK_PATTERN)
	blockRe := regexp.MustCompile("```!\\s*\\n?([\\s\\S]*?)\\n?```")
	result = blockRe.ReplaceAllStringFunc(result, func(match string) string {
		sub := blockRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cmd := strings.TrimSpace(sub[1])
		if cmd == "" {
			return match
		}
		out, err := runShellCommand(cmd)
		if err != nil {
			errs = append(errs, fmt.Sprintf("```! %s: %v", cmd, err))
			return match
		}
		return strings.TrimRight(out, "\n\r")
	})

	// Inline pattern: !`command`  (matches TS INLINE_PATTERN)
	// Go does not support lookbehind; use non-capturing (^|\s) prefix instead.
	inlineRe := regexp.MustCompile(`(?:^|\s)!` + "`([^`]+)`")
	result = inlineRe.ReplaceAllStringFunc(result, func(match string) string {
		bangBacktick := "!`"
		idx := strings.Index(match, bangBacktick)
		if idx < 0 {
			return match
		}
		prefix := match[:idx]
		cmd := strings.TrimSpace(match[idx+2 : len(match)-1])
		if cmd == "" {
			return match
		}
		out, err := runShellCommand(cmd)
		if err != nil {
			errs = append(errs, fmt.Sprintf("!`%s`: %v", cmd, err))
			return match
		}
		return prefix + strings.TrimRight(out, "\n\r")
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
