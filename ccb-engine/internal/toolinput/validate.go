// Package toolinput validates tool_use input JSON on the Go engine side before ToolRunner.Run
// (e.g. before BridgeRunner emits execute_tool to TypeScript).
package toolinput

import (
	"encoding/json"

	"goc/ccb-engine/internal/anthropic"
	"goc/internal/toolvalidator"
)

func findInputSchema(tools []anthropic.ToolDefinition, name string) any {
	for _, t := range tools {
		if t.Name == name {
			return t.InputSchema
		}
	}
	return nil
}

// ValidateAgainstTools checks input against the named tool's input_schema from tools (if present).
// Unknown tool names or missing/empty schemas are skipped (no error).
// Compilation or validation failures return a wrapped error — callers should surface them as tool_result is_error.
//
// When GO_TOOL_INPUT_VALIDATOR=zog and the tool is registered in zoglayer, Zog validates input first;
// otherwise the legacy path uses embedded JSON Schema (tools_api.json) + toolrefine.
func ValidateAgainstTools(tools []anthropic.ToolDefinition, toolName string, input json.RawMessage) error {
	rawSchema := findInputSchema(tools, toolName)
	return toolvalidator.ValidateInput(toolName, rawSchema, input)
}
