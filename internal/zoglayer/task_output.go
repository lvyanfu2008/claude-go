package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	z "github.com/Oudwins/zog"
)

var taskOutputAllowedKeys = map[string]struct{}{
	"task_id": {},
	"block":   {},
	"timeout": {},
}

type taskOutputZogInput struct {
	TaskID  string `zog:"task_id"`
	Block   *bool  `zog:"block"`
	Timeout *int   `zog:"timeout"`
}

func validateTaskOutputZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("task_output: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := taskOutputAllowedKeys[k]; !ok {
			return fmt.Errorf("task_output: unknown field %q", k)
		}
	}

	var dest taskOutputZogInput

	tRaw, ok := raw["task_id"]
	if !ok {
		return fmt.Errorf("task_output: missing required field %q", "task_id")
	}
	var tVal any
	if err := json.Unmarshal(tRaw, &tVal); err != nil {
		return fmt.Errorf("task_output: task_id: %w", err)
	}
	tStr, ok := tVal.(string)
	if !ok {
		return fmt.Errorf("task_output: task_id must be a string")
	}
	if strings.TrimSpace(tStr) == "" {
		return fmt.Errorf("task_output: task_id must be non-empty")
	}
	dest.TaskID = tStr

	if br, ok := raw["block"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("task_output: block: %w", err)
		}
		if v == nil {
			return fmt.Errorf("task_output: block cannot be null")
		}
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("task_output: block must be a boolean")
		}
		dest.Block = &b
	}

	if tr, ok := raw["timeout"]; ok {
		var tv any
		if err := json.Unmarshal(tr, &tv); err != nil {
			return fmt.Errorf("task_output: timeout: %w", err)
		}
		if tv == nil {
			return fmt.Errorf("task_output: timeout cannot be null")
		}
		f, ok := tv.(float64)
		if !ok {
			return fmt.Errorf("task_output: timeout must be a number")
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("task_output: timeout must be a finite number")
		}
		i := int(f)
		if i < 0 || i > 600000 {
			return fmt.Errorf("task_output: timeout out of range (max 600000 ms)")
		}
		dest.Timeout = &i
	}

	schema := z.Struct(z.Shape{
		"task_id": z.String().Required(),
		"block":   z.Bool().Optional(),
		"timeout": z.Int().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("task_output: zog: %v", issues)
	}
	return nil
}
