package zoglayer

import (
	"encoding/json"

	"goc/ccb-engine/bashzog"
)

func validateBashZog(input json.RawMessage) error {
	return bashzog.Validate(input)
}
