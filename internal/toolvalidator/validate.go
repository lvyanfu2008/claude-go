package toolvalidator

import (
	"encoding/json"

	"goc/internal/toolrefine"
	"goc/internal/zoglayer"
)

// ValidateInput runs tool_use input validation: when GO_TOOL_INPUT_VALIDATOR=zog and zoglayer
// registers the tool, use Zog + toolrefine; otherwise JSON Schema from schema + toolrefine (legacy).
// schema may be nil (caller should pass nil only when skipping — same as prior call sites).
func ValidateInput(toolName string, schema any, input json.RawMessage) error {
	if SkipValidation() {
		return nil
	}
	if schema == nil {
		return nil
	}
	if InputValidatorMode() == "zog" && zoglayer.Has(toolName) {
		if err := zoglayer.Validate(toolName, input); err != nil {
			return err
		}
		return toolrefine.AfterJSONSchema(toolName, input)
	}
	return ValidateLegacyJSONSchema(toolName, schema, input)
}
