package toolexecution

import (
	"encoding/json"

	"goc/internal/jsonschemavalidate"
)

// ValidateInputAgainstSchema validates instance JSON against a JSON Schema document.
// Delegates to [jsonschemavalidate.Validate] (draft from $schema when present).
func ValidateInputAgainstSchema(toolName string, schema any, input json.RawMessage) error {
	return jsonschemavalidate.Validate(toolName, schema, input)
}
