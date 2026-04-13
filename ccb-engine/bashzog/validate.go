package bashzog

import (
	"encoding/json"
	"fmt"

	z "github.com/Oudwins/zog"
)

// bashInputMinimal matches fields validated by Zog here; json.Unmarshal ignores extra keys.
type bashInputMinimal struct {
	Command string `json:"command"`
}

// Validate runs Zog validation for Bash tool_use input (required Command).
// Optional fields remain accepted in JSON but are not validated here (same minimal policy as former zoglayer/bash).
func Validate(input json.RawMessage) error {
	var dest bashInputMinimal
	if err := json.Unmarshal(input, &dest); err != nil {
		return err
	}
	s := z.Struct(z.Shape{
		"Command": z.String().Required(),
	})
	issues := s.Validate(&dest)
	if issues == nil || len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("zog: %v", issues)
}
