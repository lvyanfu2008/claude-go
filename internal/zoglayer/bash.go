package zoglayer

import (
	"encoding/json"

	"goc/ccb-engine/bashzog"
)

func validateBash(input json.RawMessage) error {
	return bashzog.Validate(input)
}
