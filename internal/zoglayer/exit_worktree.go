package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var exitWorktreeAllowedKeys = map[string]struct{}{
	"action":          {},
	"discard_changes": {},
}

func validateExitWorktreeZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("exit_worktree: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := exitWorktreeAllowedKeys[k]; !ok {
			return fmt.Errorf("exit_worktree: unknown field %q", k)
		}
	}

	aRaw, ok := raw["action"]
	if !ok {
		return fmt.Errorf("exit_worktree: missing required field %q", "action")
	}
	var aVal any
	if err := json.Unmarshal(aRaw, &aVal); err != nil {
		return fmt.Errorf("exit_worktree: action: %w", err)
	}
	aStr, ok := aVal.(string)
	if !ok {
		return fmt.Errorf("exit_worktree: action must be a string")
	}
	if aStr != "keep" && aStr != "remove" {
		return fmt.Errorf("exit_worktree: action must be one of [keep, remove]")
	}

	if br, ok := raw["discard_changes"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("exit_worktree: discard_changes: %w", err)
		}
		if v == nil {
			return fmt.Errorf("exit_worktree: discard_changes cannot be null")
		}
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("exit_worktree: discard_changes must be a boolean")
		}
	}

	return nil
}
