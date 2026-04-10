// Package bashprepare implements Phase-1 stdin/stdout JSON prepare for bash-mode input (no shell execution).
package bashprepare

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxCommandRunes = 512 * 1024

// StdinRequest matches the TS bridge payload.
type StdinRequest struct {
	Input string `json:"input"`
	Shell string `json:"shell,omitempty"` // "bash" | "powershell" — reserved for future branches
}

// Reject when the command must not run.
type Reject struct {
	Reason string `json:"reason"`
}

// Result is written to stdout JSON (encode omits empty fields).
type Result struct {
	Command   string   `json:"command"`
	Warnings  []string `json:"warnings,omitempty"`
	Reject    *Reject  `json:"reject,omitempty"`
	ShellHint string   `json:"shellHint,omitempty"` // echo of request shell for debugging
}

// Prepare validates and normalizes input without executing the shell.
func Prepare(req StdinRequest) Result {
	shell := strings.TrimSpace(strings.ToLower(req.Shell))
	if shell == "" {
		shell = "bash"
	}

	raw := req.Input
	if !utf8.ValidString(raw) {
		return Result{
			Reject: &Reject{Reason: "Input is not valid UTF-8"},
		}
	}
	if strings.ContainsRune(raw, '\x00') {
		return Result{
			Reject: &Reject{Reason: "Input contains null bytes"},
		}
	}

	cmd := strings.TrimSpace(raw)
	if cmd == "" {
		return Result{
			Reject: &Reject{Reason: "Empty command after trimming"},
		}
	}

	var warnings []string
	if utf8.RuneCountInString(cmd) > maxCommandRunes {
		warnings = append(warnings, fmt.Sprintf("Command exceeds %d Unicode scalars (very long input)", maxCommandRunes))
	}

	out := Result{
		Command:  cmd,
		Warnings: warnings,
	}
	if shell != "" && shell != "bash" {
		out.ShellHint = shell
	}
	return out
}

// Run reads JSON from stdin and writes Result JSON to stdout; returns error for stderr / exit 1.
func Run(stdin []byte) ([]byte, error) {
	var req StdinRequest
	if len(stdin) == 0 {
		return nil, fmt.Errorf("empty stdin")
	}
	if err := json.Unmarshal(stdin, &req); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	res := Prepare(req)
	enc, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return enc, nil
}
