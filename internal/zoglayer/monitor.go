package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var monitorAllowedKeys = map[string]struct{}{
	"command":     {},
	"description": {},
}

type monitorZogInput struct {
	Command     string `zog:"command"`
	Description string `zog:"description"`
}

func validateMonitorZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("monitor: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := monitorAllowedKeys[k]; !ok {
			return fmt.Errorf("monitor: unknown field %q", k)
		}
	}

	var dest monitorZogInput

	cRaw, ok := raw["command"]
	if !ok {
		return fmt.Errorf("monitor: missing required field %q", "command")
	}
	var cVal any
	if err := json.Unmarshal(cRaw, &cVal); err != nil {
		return fmt.Errorf("monitor: command: %w", err)
	}
	cStr, ok := cVal.(string)
	if !ok {
		return fmt.Errorf("monitor: command must be a string")
	}
	if strings.TrimSpace(cStr) == "" {
		return fmt.Errorf("monitor: command must be non-empty")
	}
	dest.Command = cStr

	dRaw, ok := raw["description"]
	if !ok {
		return fmt.Errorf("monitor: missing required field %q", "description")
	}
	var dVal any
	if err := json.Unmarshal(dRaw, &dVal); err != nil {
		return fmt.Errorf("monitor: description: %w", err)
	}
	dStr, ok := dVal.(string)
	if !ok {
		return fmt.Errorf("monitor: description must be a string")
	}
	dest.Description = dStr

	schema := z.Struct(z.Shape{
		"command":     z.String().Required(),
		"description": z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("monitor: zog: %v", issues)
	}
	return nil
}
