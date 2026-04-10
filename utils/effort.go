// Package utils mirrors src/utils/effort.ts (effort level / value for API).
// Path rule: src/… in TS ↔ go/… in Go.
package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EffortLevel mirrors src/entrypoints/sdk/runtimeTypes.ts EffortLevel.
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortMax    EffortLevel = "max"
)

// EffortValue mirrors src/utils/effort.ts EffortValue (EffortLevel | number).
type EffortValue struct {
	v any // EffortLevel string or float64
}

// EffortFromLevel builds an EffortValue from a named level.
func EffortFromLevel(l EffortLevel) EffortValue {
	return EffortValue{v: string(l)}
}

// EffortFromNumber builds an EffortValue from a numeric override.
func EffortFromNumber(n float64) EffortValue {
	return EffortValue{v: n}
}

// Level returns the named level when the JSON value was a string level.
func (e EffortValue) Level() (EffortLevel, bool) {
	s, ok := e.v.(string)
	if !ok {
		return "", false
	}
	return EffortLevel(s), true
}

// Number returns the numeric value when the JSON value was a number.
func (e EffortValue) Number() (float64, bool) {
	n, ok := e.v.(float64)
	if ok {
		return n, true
	}
	// json.Unmarshal uses float64 for all numbers; guard for json.Number if ever used
	return 0, false
}

// MarshalJSON encodes either a string level or a number, matching TS unions.
func (e EffortValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.v)
}

// UnmarshalJSON decodes a JSON string (level) or number.
func (e *EffortValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		e.v = nil
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		e.v = n
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	e.v = s
	return nil
}

// String describes the value for debugging (not TS API).
func (e EffortValue) String() string {
	if e.v == nil {
		return "<nil>"
	}
	switch x := e.v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprint(x)
	default:
		return fmt.Sprint(x)
	}
}

// ParseEffortValueYAML parses YAML frontmatter `effort` (string level or number), mirroring src/utils/effort.ts parseEffortValue.
func ParseEffortValueYAML(v interface{}) (EffortValue, bool) {
	if v == nil {
		return EffortValue{}, false
	}
	switch t := v.(type) {
	case int:
		return EffortFromNumber(float64(t)), true
	case int64:
		return EffortFromNumber(float64(t)), true
	case float64:
		return EffortFromNumber(t), true
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "" {
			return EffortValue{}, false
		}
		if isEffortLevel(s) {
			return EffortFromLevel(EffortLevel(s)), true
		}
		return EffortValue{}, false
	default:
		return EffortValue{}, false
	}
}

func isEffortLevel(s string) bool {
	switch EffortLevel(s) {
	case EffortLow, EffortMedium, EffortHigh, EffortMax:
		return true
	default:
		return false
	}
}
