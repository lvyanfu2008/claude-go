package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var taskStopAllowedKeys = map[string]struct{}{
	"task_id": {},
	"shell_id": {},
}

type taskStopZogInput struct {
	TaskID  *string `zog:"task_id"`
	ShellID *string `zog:"shell_id"`
}

func validateTaskStopZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("task_stop: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := taskStopAllowedKeys[k]; !ok {
			return fmt.Errorf("task_stop: unknown field %q", k)
		}
	}

	var dest taskStopZogInput
	if err := parseZogStringField(raw, "task_id", &dest.TaskID); err != nil {
		return fmt.Errorf("task_stop: %w", err)
	}
	if err := parseZogStringField(raw, "shell_id", &dest.ShellID); err != nil {
		return fmt.Errorf("task_stop: %w", err)
	}

	schema := z.Struct(z.Shape{
		"task_id":  z.String().Optional(),
		"shell_id": z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("task_stop: zog: %v", issues)
	}
	return nil
}
