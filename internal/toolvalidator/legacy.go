package toolvalidator

import (
	"encoding/json"

	"goc/internal/jsonschemavalidate"
	"goc/internal/toolrefine"
)

// ValidateLegacyJSONSchema runs the historical path: JSON Schema (from TS export shape) then toolrefine.
func ValidateLegacyJSONSchema(toolName string, schema any, input json.RawMessage) error {
	if err := jsonschemavalidate.Validate(toolName, schema, input); err != nil {
		return err
	}
	return toolrefine.AfterJSONSchema(toolName, input)
}
