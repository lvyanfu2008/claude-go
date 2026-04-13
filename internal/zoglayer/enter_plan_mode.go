package zoglayer

import (
	"encoding/json"

	z "github.com/Oudwins/zog"
)

type enterPlanModeInput struct{}

// enterPlanModeSchema mirrors TS EnterPlanMode: empty object (no required keys).
// Note: Zog may not enforce additionalProperties:false; strict shape parity uses the jsonschema path.
func validateEnterPlanMode(input json.RawMessage) error {
	var dest enterPlanModeInput
	if err := json.Unmarshal(input, &dest); err != nil {
		return err
	}
	s := z.Struct(z.Shape{})
	issues := s.Validate(&dest)
	return issuesToErr(issues)
}
