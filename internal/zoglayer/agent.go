package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var agentAllowedKeys = map[string]struct{}{
	"description":       {},
	"prompt":            {},
	"subagent_type":     {},
	"model":             {},
	"name":              {},
	"team_name":         {},
	"mode":              {},
	"isolation":         {},
	"cwd":               {},
	"run_in_background": {},
}

var agentValidModels = map[string]bool{
	"sonnet": true,
	"opus":   true,
	"haiku":  true,
}

var agentValidModes = map[string]bool{
	"acceptEdits":       true,
	"bypassPermissions": true,
	"default":           true,
	"dontAsk":           true,
	"plan":              true,
	"auto":              true,
}

var agentValidIsolation = map[string]bool{
	"worktree": true,
}

func validateAgentZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("agent: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := agentAllowedKeys[k]; !ok {
			return fmt.Errorf("agent: unknown field %q", k)
		}
	}

	dRaw, ok := raw["description"]
	if !ok {
		return fmt.Errorf("agent: missing required field %q", "description")
	}
	var dVal any
	if err := json.Unmarshal(dRaw, &dVal); err != nil {
		return fmt.Errorf("agent: description: %w", err)
	}
	dStr, ok := dVal.(string)
	if !ok {
		return fmt.Errorf("agent: description must be a string")
	}
	if strings.TrimSpace(dStr) == "" {
		return fmt.Errorf("agent: description must be non-empty")
	}

	pRaw, ok := raw["prompt"]
	if !ok {
		return fmt.Errorf("agent: missing required field %q", "prompt")
	}
	var pVal any
	if err := json.Unmarshal(pRaw, &pVal); err != nil {
		return fmt.Errorf("agent: prompt: %w", err)
	}
	pStr, ok := pVal.(string)
	if !ok {
		return fmt.Errorf("agent: prompt must be a string")
	}
	if strings.TrimSpace(pStr) == "" {
		return fmt.Errorf("agent: prompt must be non-empty")
	}

	// Optional enum: subagent_type
	if br, ok := raw["subagent_type"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: subagent_type: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: subagent_type cannot be null")
		}
		if _, ok := v.(string); !ok {
			return fmt.Errorf("agent: subagent_type must be a string")
		}
	}

	// Optional enum: model
	if br, ok := raw["model"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: model: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: model cannot be null")
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("agent: model must be a string")
		}
		if !agentValidModels[s] {
			return fmt.Errorf("agent: model must be one of [sonnet, opus, haiku]")
		}
	}

	// Optional string: name
	if br, ok := raw["name"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: name: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("agent: name must be a string")
			}
		}
	}

	// Optional string: team_name
	if br, ok := raw["team_name"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: team_name: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: team_name cannot be null")
		}
		if _, ok := v.(string); !ok {
			return fmt.Errorf("agent: team_name must be a string")
		}
	}

	// Optional enum: mode
	if br, ok := raw["mode"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: mode: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: mode cannot be null")
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("agent: mode must be a string")
		}
		if !agentValidModes[s] {
			return fmt.Errorf("agent: mode must be one of [acceptEdits, bypassPermissions, default, dontAsk, plan, auto]")
		}
	}

	// Optional enum: isolation
	if br, ok := raw["isolation"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: isolation: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: isolation cannot be null")
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("agent: isolation must be a string")
		}
		if !agentValidIsolation[s] {
			return fmt.Errorf("agent: isolation must be one of [worktree]")
		}
	}

	// Optional string: cwd
	if br, ok := raw["cwd"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: cwd: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: cwd cannot be null")
		}
		if _, ok := v.(string); !ok {
			return fmt.Errorf("agent: cwd must be a string")
		}
	}

	// Optional bool: run_in_background
	if br, ok := raw["run_in_background"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("agent: run_in_background: %w", err)
		}
		if v == nil {
			return fmt.Errorf("agent: run_in_background cannot be null")
		}
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("agent: run_in_background must be a boolean")
		}
	}

	return nil
}
