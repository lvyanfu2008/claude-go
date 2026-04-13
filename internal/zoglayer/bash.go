package zoglayer

import (
	"encoding/json"

	z "github.com/Oudwins/zog"
)

// bashInputMinimal holds fields Zog validates here; encoding/json ignores extra JSON keys (timeout, …).
type bashInputMinimal struct {
	Command string `json:"command"`
}

// bashInputSchema mirrors TS Bash tool_use input for the required "command" field.
// Optional fields are accepted by JSON unmarshal but not validated by this minimal Zog schema
// (full shape parity remains on the jsonschema path).
func validateBash(input json.RawMessage) error {
	var dest bashInputMinimal
	if err := json.Unmarshal(input, &dest); err != nil {
		return err
	}
	s := z.Struct(z.Shape{
		"Command": z.String().Required(),
	})
	issues := s.Validate(&dest)
	return issuesToErr(issues)
}
