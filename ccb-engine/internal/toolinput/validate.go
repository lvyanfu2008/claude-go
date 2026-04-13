// Package toolinput validates tool_use input JSON on the Go engine side before ToolRunner.Run
// (e.g. before BridgeRunner emits execute_tool to TypeScript).
package toolinput

import (
	"encoding/json"
	"os"
	"strings"

	"goc/ccb-engine/internal/anthropic"
	"goc/internal/jsonschemavalidate"
	"goc/internal/toolrefine"
)

// SkipValidation returns true when CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA=1 (escape hatch).
func SkipValidation() bool {
	return strings.TrimSpace(os.Getenv("CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA")) == "1"
}

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
func ValidateAgainstTools(tools []anthropic.ToolDefinition, toolName string, input json.RawMessage) error {
	if SkipValidation() {
		return nil
	}
	rawSchema := findInputSchema(tools, toolName)
	if rawSchema == nil {
		return nil
	}

	if err := jsonschemavalidate.Validate(toolName, rawSchema, input); err != nil {
		return err
	}
	return toolrefine.AfterJSONSchema(toolName, input)
}
