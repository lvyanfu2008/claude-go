package toolexecution

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidateInputAgainstSchema validates instance JSON against a JSON Schema document (Draft 7).
// Mirrors the core of ccb-engine/toolinput.ValidateAgainstTools without importing internal packages.
func ValidateInputAgainstSchema(toolName string, schema any, input json.RawMessage) error {
	if schema == nil {
		return nil
	}
	switch v := schema.(type) {
	case bool:
		if !v {
			return fmt.Errorf("tool %q input_schema is false (no inputs accepted)", toolName)
		}
		return nil
	}
	schemaBytes, err := json.Marshal(schema)
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
	loc := "https://toolexecution.local/tool-input-schema/" + toolName
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
