package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var enterWorktreeAllowedKeys = map[string]struct{}{
	"name": {},
}

func validateEnterWorktreeZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("enter_worktree: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := enterWorktreeAllowedKeys[k]; !ok {
			return fmt.Errorf("enter_worktree: unknown field %q", k)
		}
	}
	if br, ok := raw["name"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("enter_worktree: name: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("enter_worktree: name must be a string")
			}
		}
	}
	return nil
}
