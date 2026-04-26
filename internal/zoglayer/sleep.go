package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	z "github.com/Oudwins/zog"
)

var sleepAllowedKeys = map[string]struct{}{
	"duration_seconds": {},
}

type sleepZogInput struct {
	DurationSeconds float64 `zog:"duration_seconds"`
}

func validateSleepZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("sleep: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := sleepAllowedKeys[k]; !ok {
			return fmt.Errorf("sleep: unknown field %q", k)
		}
	}

	dRaw, ok := raw["duration_seconds"]
	if !ok {
		return fmt.Errorf("sleep: missing required field %q", "duration_seconds")
	}
	var dVal any
	if err := json.Unmarshal(dRaw, &dVal); err != nil {
		return fmt.Errorf("sleep: duration_seconds: %w", err)
	}
	if dVal == nil {
		return fmt.Errorf("sleep: duration_seconds cannot be null")
	}
	f, ok := dVal.(float64)
	if !ok {
		return fmt.Errorf("sleep: duration_seconds must be a number")
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("sleep: duration_seconds must be a finite number")
	}

	dest := sleepZogInput{DurationSeconds: f}
	schema := z.Struct(z.Shape{
		"duration_seconds": z.Float64().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("sleep: zog: %v", issues)
	}
	return nil
}
