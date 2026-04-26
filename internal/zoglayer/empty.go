package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateEmptyObjectZog rejects any non-empty object or unknown fields.
// Use for tools with an empty input schema (no properties).
func validateEmptyObjectZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	if len(raw) > 0 {
		return fmt.Errorf("empty: unexpected fields")
	}
	return nil
}
