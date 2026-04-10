// Package slashprepare implements Phase-1 stdin/stdout JSON prepare for slash input (parse only; no command execution).
package slashprepare

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Matches TS `processSlashCommand` user-facing copy when parse fails.
const errSlashForm = "Commands are in the form `/command [args]`"

const maxInputRunes = 512 * 1024

// StdinRequest matches the TS bridge payload.
type StdinRequest struct {
	Input string `json:"input"`
}

// Reject when the line cannot be treated as a slash command.
type Reject struct {
	Reason string `json:"reason"`
}

// Result is written to stdout JSON. On success mirrors TS ParsedSlashCommand; `args` is always present (may be "") so TS can parse reliably.
type Result struct {
	CommandName string   `json:"commandName"`
	Args        string   `json:"args"`
	IsMcp       bool     `json:"isMcp"`
	Warnings    []string `json:"warnings,omitempty"`
	Reject      *Reject  `json:"reject,omitempty"`
}

// Prepare validates and parses input using the same rules as TS `parseSlashCommand` (split on ASCII space only).
func Prepare(req StdinRequest) Result {
	raw := req.Input
	if !utf8.ValidString(raw) {
		return Result{Reject: &Reject{Reason: "Input is not valid UTF-8"}}
	}
	if strings.ContainsRune(raw, '\x00') {
		return Result{Reject: &Reject{Reason: "Input contains null bytes"}}
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return Result{Reject: &Reject{Reason: errSlashForm}}
	}

	withoutSlash := trimmed[1:]
	words := strings.Split(withoutSlash, " ")
	if len(words) == 0 || words[0] == "" {
		return Result{Reject: &Reject{Reason: errSlashForm}}
	}

	commandName := words[0]
	isMcp := false
	argsStart := 1
	if len(words) > 1 && words[1] == "(MCP)" {
		commandName = commandName + " (MCP)"
		isMcp = true
		argsStart = 2
	}
	args := strings.Join(words[argsStart:], " ")

	var warnings []string
	if utf8.RuneCountInString(trimmed) > maxInputRunes {
		warnings = append(warnings, fmt.Sprintf("Input exceeds %d Unicode scalars (very long line)", maxInputRunes))
	}

	return Result{
		CommandName: commandName,
		Args:        args,
		IsMcp:       isMcp,
		Warnings:    warnings,
	}
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
