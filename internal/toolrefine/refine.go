package toolrefine

import "encoding/json"

// AfterJSONSchema runs Zod-only rules that are not represented in the API JSON Schema.
func AfterJSONSchema(toolName string, input json.RawMessage) error {
	switch toolName {
	case "AskUserQuestion":
		return ValidateAskUserQuestionUniqueness(input)
	default:
		return nil
	}
}
