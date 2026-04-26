package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	z "github.com/Oudwins/zog"
)

var powerShellAllowedKeys = map[string]struct{}{
	"command":           {},
	"timeout":           {},
	"run_in_background": {},
}

type powerShellZogInput struct {
	Command         string `zog:"command"`
	Timeout         *int   `zog:"timeout"`
	RunInBackground *bool  `zog:"run_in_background"`
}

func validatePowerShellZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("powershell: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := powerShellAllowedKeys[k]; !ok {
			return fmt.Errorf("powershell: unknown field %q", k)
		}
	}

	var dest powerShellZogInput

	cRaw, ok := raw["command"]
	if !ok {
		return fmt.Errorf("powershell: missing required field %q", "command")
	}
	var cVal any
	if err := json.Unmarshal(cRaw, &cVal); err != nil {
		return fmt.Errorf("powershell: command: %w", err)
	}
	cStr, ok := cVal.(string)
	if !ok {
		return fmt.Errorf("powershell: command must be a string")
	}
	if strings.TrimSpace(cStr) == "" {
		return fmt.Errorf("powershell: command must be non-empty")
	}
	dest.Command = cStr

	if tr, ok := raw["timeout"]; ok {
		var tv any
		if err := json.Unmarshal(tr, &tv); err != nil {
			return fmt.Errorf("powershell: timeout: %w", err)
		}
		if tv == nil {
			return fmt.Errorf("powershell: timeout cannot be null")
		}
		f, ok := tv.(float64)
		if !ok {
			return fmt.Errorf("powershell: timeout must be a number")
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("powershell: timeout must be a finite number")
		}
		i := int(f)
		dest.Timeout = &i
	}

	if br, ok := raw["run_in_background"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("powershell: run_in_background: %w", err)
		}
		if v == nil {
			return fmt.Errorf("powershell: run_in_background cannot be null")
		}
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("powershell: run_in_background must be a boolean")
		}
		dest.RunInBackground = &b
	}

	schema := z.Struct(z.Shape{
		"command":           z.String().Required(),
		"timeout":           z.Int().Optional(),
		"run_in_background": z.Bool().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("powershell: zog: %v", issues)
	}
	return nil
}
