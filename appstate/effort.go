package appstate

import (
	"bytes"
	"encoding/json"
)

// Effort levels (src/utils/effort.ts EFFORT_LEVELS / EffortLevel).
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortMax    EffortLevel = "max"
)

// EffortValue mirrors src/utils/effort.ts EffortValue (EffortLevel | number). JSON is a string or a number.
type EffortValue struct {
	Level  EffortLevel `json:"-"`
	Number float64     `json:"-"`
	IsNum  bool        `json:"-"`
}

// MarshalJSON emits either a string level or a JSON number.
func (e EffortValue) MarshalJSON() ([]byte, error) {
	if e.IsNum {
		return json.Marshal(e.Number)
	}
	if e.Level != "" {
		return json.Marshal(string(e.Level))
	}
	return []byte("null"), nil
}

// UnmarshalJSON accepts string ("low", …) or number (TS numeric effort).
func (e *EffortValue) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*e = EffortValue{}
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*e = EffortValue{Level: EffortLevel(s), IsNum: false}
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*e = EffortValue{Number: n, IsNum: true}
	return nil
}
