package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var exitPlanModeAllowedKeys = map[string]struct{}{
	"allowedPrompts": {},
}

type exitPlanModePrompt struct {
	Tool   string `json:"tool"`
	Prompt string `json:"prompt"`
}

func validateExitPlanModeZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := exitPlanModeAllowedKeys[k]; !ok {
			return fmt.Errorf("exit_plan_mode: unknown field %q", k)
		}
	}
	if ar, ok := raw["allowedPrompts"]; ok {
		var arr []json.RawMessage
		if err := json.Unmarshal(ar, &arr); err != nil {
			return fmt.Errorf("exit_plan_mode: allowedPrompts must be an array: %w", err)
		}
		for i, item := range arr {
			var p exitPlanModePrompt
			if err := json.Unmarshal(item, &p); err != nil {
				return fmt.Errorf("exit_plan_mode: allowedPrompts[%d]: %w", i, err)
			}
			if strings.TrimSpace(p.Tool) == "" {
				return fmt.Errorf("exit_plan_mode: allowedPrompts[%d]: tool must be non-empty", i)
			}
			if p.Tool != "Bash" {
				return fmt.Errorf("exit_plan_mode: allowedPrompts[%d]: tool must be \"Bash\"", i)
			}
			if strings.TrimSpace(p.Prompt) == "" {
				return fmt.Errorf("exit_plan_mode: allowedPrompts[%d]: prompt must be non-empty", i)
			}
		}
	}
	return nil
}
