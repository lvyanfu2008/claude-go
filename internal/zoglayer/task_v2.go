package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var taskCreateAllowedKeys = map[string]struct{}{
	"type":        {},
	"subject":     {},
	"description": {},
	"activeForm":  {},
	"status":      {},
	"owner":       {},
	"blocks":      {},
	"blockedBy":   {},
	"metadata":    {},
}

func validateTaskCreateZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("task_create: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := taskCreateAllowedKeys[k]; !ok {
			return fmt.Errorf("task_create: unknown field %q", k)
		}
	}
	sRaw, ok := raw["subject"]
	if !ok {
		return fmt.Errorf("task_create: missing required field %q", "subject")
	}
	var sVal any
	if err := json.Unmarshal(sRaw, &sVal); err != nil {
		return fmt.Errorf("task_create: subject: %w", err)
	}
	sStr, ok := sVal.(string)
	if !ok {
		return fmt.Errorf("task_create: subject must be a string")
	}
	if strings.TrimSpace(sStr) == "" {
		return fmt.Errorf("task_create: subject must be non-empty")
	}
	// description, activeForm: optional strings
	if br, ok := raw["description"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("task_create: description: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("task_create: description must be a string")
			}
		}
	}
	if br, ok := raw["activeForm"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("task_create: activeForm: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("task_create: activeForm must be a string")
			}
		}
	}
	// metadata: optional object
	if br, ok := raw["metadata"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("task_create: metadata: %w", err)
		}
		if v != nil {
			if _, ok := v.(map[string]any); !ok {
				return fmt.Errorf("task_create: metadata must be an object")
			}
		}
	}
	// type, status, owner: optional strings
	for _, k := range []string{"type", "status", "owner"} {
		if br, ok := raw[k]; ok {
			var v any
			if err := json.Unmarshal(br, &v); err != nil {
				return fmt.Errorf("task_create: %s: %w", k, err)
			}
			if v != nil {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("task_create: %s must be a string", k)
				}
			}
		}
	}
	// blocks, blockedBy: optional string arrays
	for _, k := range []string{"blocks", "blockedBy"} {
		if br, ok := raw[k]; ok {
			var arr []any
			if err := json.Unmarshal(br, &arr); err != nil {
				return fmt.Errorf("task_create: %s must be an array of strings: %w", k, err)
			}
			for i, v := range arr {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("task_create: %s[%d] must be a string", k, i)
				}
			}
		}
	}
	return nil
}

var taskGetAllowedKeys = map[string]struct{}{
	"taskId": {},
}

func validateTaskGetZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("task_get: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := taskGetAllowedKeys[k]; !ok {
			return fmt.Errorf("task_get: unknown field %q", k)
		}
	}
	idRaw, ok := raw["taskId"]
	if !ok {
		return fmt.Errorf("task_get: missing required field %q", "taskId")
	}
	var idVal any
	if err := json.Unmarshal(idRaw, &idVal); err != nil {
		return fmt.Errorf("task_get: taskId: %w", err)
	}
	idStr, ok := idVal.(string)
	if !ok {
		return fmt.Errorf("task_get: taskId must be a string")
	}
	if strings.TrimSpace(idStr) == "" {
		return fmt.Errorf("task_get: taskId must be non-empty")
	}
	return nil
}

func validateTaskListZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		return fmt.Errorf("task_list: unknown field %q", k)
	}
	return nil
}

var taskUpdateAllowedKeys = map[string]struct{}{
	"taskId":       {},
	"subject":      {},
	"description":  {},
	"activeForm":   {},
	"status":       {},
	"owner":        {},
	"addBlocks":    {},
	"addBlockedBy": {},
	"metadata":     {},
}

func validateTaskUpdateZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("task_update: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := taskUpdateAllowedKeys[k]; !ok {
			return fmt.Errorf("task_update: unknown field %q", k)
		}
	}
	idRaw, ok := raw["taskId"]
	if !ok {
		return fmt.Errorf("task_update: missing required field %q", "taskId")
	}
	var idVal any
	if err := json.Unmarshal(idRaw, &idVal); err != nil {
		return fmt.Errorf("task_update: taskId: %w", err)
	}
	idStr, ok := idVal.(string)
	if !ok {
		return fmt.Errorf("task_update: taskId must be a string")
	}
	if strings.TrimSpace(idStr) == "" {
		return fmt.Errorf("task_update: taskId must be non-empty")
	}
	// Optional string fields
	for _, k := range []string{"subject", "description", "activeForm", "status", "owner"} {
		if br, ok := raw[k]; ok {
			var v any
			if err := json.Unmarshal(br, &v); err != nil {
				return fmt.Errorf("task_update: %s: %w", k, err)
			}
			if v != nil {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("task_update: %s must be a string", k)
				}
				if k == "status" {
					switch s {
					case "pending", "in_progress", "completed", "failed", "killed", "deleted":
					default:
						return fmt.Errorf("task_update: invalid status %q (must be pending, in_progress, completed, failed, killed, or deleted)", s)
					}
				}
			}
		}
	}
	// Optional string array fields
	for _, k := range []string{"addBlocks", "addBlockedBy"} {
		if br, ok := raw[k]; ok {
			var arr []any
			if err := json.Unmarshal(br, &arr); err != nil {
				return fmt.Errorf("task_update: %s must be an array of strings: %w", k, err)
			}
			for i, v := range arr {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("task_update: %s[%d] must be a string", k, i)
				}
			}
		}
	}
	// metadata: optional object
	if br, ok := raw["metadata"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("task_update: metadata: %w", err)
		}
		if v != nil {
			if _, ok := v.(map[string]any); !ok {
				return fmt.Errorf("task_update: metadata must be an object")
			}
		}
	}
	return nil
}
