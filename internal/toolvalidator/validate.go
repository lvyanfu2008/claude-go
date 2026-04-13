package toolvalidator

import (
	"encoding/json"

	"goc/internal/toolrefine"
	"goc/internal/zoglayer"
)

// ValidateInput runs tool_use input validation: when GO_TOOL_INPUT_VALIDATOR=zog and zoglayer
// registers the tool, use Zog + toolrefine; otherwise JSON Schema from schema + toolrefine (legacy).
// For zoglayer tools in zog mode, schema may be nil (callers do not need embed tools_api input_schema).
func ValidateInput(toolName string, schema any, input json.RawMessage) error {
	if InputValidatorMode() == "zog" && zoglayer.Has(toolName) {
		if err := zoglayer.Validate(toolName, input); err != nil {
			return err
		}
		return toolrefine.AfterJSONSchema(toolName, input)
	}
	if schema == nil {
		return nil
	}
	return ValidateLegacyJSONSchema(toolName, schema, input)
}
