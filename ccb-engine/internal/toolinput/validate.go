// Package toolinput validates tool_use input JSON on the Go engine side before ToolRunner.Run
// (e.g. before BridgeRunner emits execute_tool to TypeScript).
package toolinput

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"goc/ccb-engine/internal/anthropic"
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

	switch v := rawSchema.(type) {
	case bool:
		if !v {
			return fmt.Errorf("tool %q input_schema is false (no inputs accepted)", toolName)
		}
		return nil
	}

	schemaBytes, err := json.Marshal(rawSchema)
	if err != nil {
		return fmt.Errorf("tool %q input_schema: marshal: %w", toolName, err)
	}
	trimmed := bytes.TrimSpace(schemaBytes)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return fmt.Errorf("tool %q input_schema: parse: %w", toolName, err)
	}

	instReader := bytes.NewReader(input)
	if len(bytes.TrimSpace(input)) == 0 {
		instReader = bytes.NewReader([]byte("{}"))
	}
	inst, err := jsonschema.UnmarshalJSON(instReader)
	if err != nil {
		return fmt.Errorf("tool %q input: %w", toolName, err)
	}

	loc := "https://ccb-engine.local/tool-input-schema/" + toolName
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft7)
	if err := c.AddResource(loc, schemaDoc); err != nil {
		return fmt.Errorf("tool %q input_schema: add resource: %w", toolName, err)
	}
	sch, err := c.Compile(loc)
	if err != nil {
		return fmt.Errorf("tool %q input_schema: compile: %w", toolName, err)
	}
	if err := sch.Validate(inst); err != nil {
		return fmt.Errorf("tool %q input: %w", toolName, err)
	}
	return nil
}
