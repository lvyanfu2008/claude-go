package toolpool

import (
	"encoding/json"

	"goc/types"
)

// anthropicToolDefinition matches tool objects in tools_api.json (name, description, input_schema).
type anthropicToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// MarshalToolsAPIDocumentDefinitions encodes tools for ToolUseContext.Options.Tools (Anthropic-style array).
func MarshalToolsAPIDocumentDefinitions(tools []types.ToolSpec) (json.RawMessage, error) {
	defs := make([]anthropicToolDefinition, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, anthropicToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputJSONSchema,
		})
	}
	return json.Marshal(defs)
}
