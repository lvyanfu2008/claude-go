package zoglayer

import (
	"encoding/json"
)

// validators holds tools with a Zog-based input path (GO_TOOL_INPUT_VALIDATOR=zog).
// Tools not listed here fall back to JSON Schema + toolrefine in toolvalidator.
var validators = map[string]func(json.RawMessage) error{
	"Bash":           validateBash,
	"EnterPlanMode":  validateEnterPlanMode,
}

// Has reports whether toolName uses the Zog validator when Zog mode is on.
func Has(toolName string) bool {
	_, ok := validators[toolName]
	return ok
}

// Validate runs the Zog schema for toolName. Caller must ensure Has(toolName).
func Validate(toolName string, input json.RawMessage) error {
	f := validators[toolName]
	if f == nil {
		return nil
	}
	return f(input)
}
