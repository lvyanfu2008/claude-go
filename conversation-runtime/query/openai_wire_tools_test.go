package query

import (
	"encoding/json"
	"testing"
)

func TestSanitizeJsonSchema_StripsDollarSchema(t *testing.T) {
	input := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}
	result := sanitizeJsonSchema(input)
	if _, ok := result["$schema"]; ok {
		t.Error("$schema should be stripped")
	}
}

func TestSanitizeJsonSchema_StripsPropertyNames(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"propertyNames": map[string]any{
			"type": "string",
		},
		"additionalProperties": map[string]any{"type": "string"},
	}
	result := sanitizeJsonSchema(input)
	if _, ok := result["propertyNames"]; ok {
		t.Error("propertyNames should be stripped")
	}
}

func TestSanitizeJsonSchema_ConstToEnum(t *testing.T) {
	input := map[string]any{
		"type":  "string",
		"const": "read",
	}
	result := sanitizeJsonSchema(input)
	if _, ok := result["const"]; ok {
		t.Error("const should be converted to enum")
	}
	enum, ok := result["enum"].([]any)
	if !ok || len(enum) != 1 || enum[0] != "read" {
		t.Errorf("expected enum: [read], got %v", result["enum"])
	}
}

func TestSanitizeJsonSchema_RecursivelyProcessesNestedProperties(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{
							"type":  "string",
							"const": "hello",
						},
					},
				},
			},
		},
	}
	result := sanitizeJsonSchema(input)
	// Navigate to the nested label schema
	props, _ := result["properties"].(map[string]any)
	questions, _ := props["questions"].(map[string]any)
	items, _ := questions["items"].(map[string]any)
	itemProps, _ := items["properties"].(map[string]any)
	label, _ := itemProps["label"].(map[string]any)
	if _, ok := label["const"]; ok {
		t.Error("nested const should be converted to enum")
	}
	if _, ok := label["enum"]; !ok {
		t.Error("nested const should become enum")
	}
}

func TestSanitizeJsonSchema_RecursivelyProcessesAdditionalProperties(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"additionalProperties": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"preview": map[string]any{
					"type":  "string",
					"const": "x",
				},
			},
		},
	}
	result := sanitizeJsonSchema(input)
	addProps, _ := result["additionalProperties"].(map[string]any)
	props, _ := addProps["properties"].(map[string]any)
	preview, _ := props["preview"].(map[string]any)
	if _, ok := preview["const"]; ok {
		t.Error("nested const in additionalProperties should be converted to enum")
	}
}

func TestSanitizeJsonSchema_RecursivelyProcessesAnyOf(t *testing.T) {
	input := map[string]any{
		"anyOf": []any{
			map[string]any{"type": "string", "const": "a"},
			map[string]any{"type": "string", "const": "b"},
		},
	}
	result := sanitizeJsonSchema(input)
	anyOf, _ := result["anyOf"].([]any)
	for _, item := range anyOf {
		m := item.(map[string]any)
		if _, ok := m["const"]; ok {
			t.Error("const in anyOf should be converted to enum")
		}
	}
}

func TestSanitizeJsonSchema_NilInput(t *testing.T) {
	result := sanitizeJsonSchema(nil)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestAnthropicToolsWireToOpenAI_SanitizesParameters(t *testing.T) {
	// Build an Anthropic-format tool with AskUserQuestion-like schema
	// that has $schema, propertyNames, and const.
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{"type": "string"},
						"header":   map[string]any{"type": "string"},
						"options": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"label":       map[string]any{"type": "string"},
									"description": map[string]any{"type": "string"},
									"preview":     map[string]any{"type": "string"},
								},
								"required": []string{"label", "description"},
							},
						},
						"multiSelect": map[string]any{"type": "boolean", "default": false},
					},
					"required": []string{"question", "header", "options", "multiSelect"},
				},
			},
			"answers": map[string]any{
				"type": "object",
				"propertyNames": map[string]any{
					"type": "string",
				},
				"additionalProperties": map[string]any{"type": "string"},
			},
			"annotations": map[string]any{
				"type": "object",
				"propertyNames": map[string]any{
					"type": "string",
				},
				"additionalProperties": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"preview": map[string]any{"type": "string"},
						"notes":   map[string]any{"type": "string"},
					},
				},
			},
		},
		"required": []string{"questions"},
	}

	tool := map[string]any{
		"name":         "AskUserQuestion",
		"description":  "Ask the user clarifying multiple-choice questions during execution.",
		"input_schema": schema,
	}

	toolsJSON, err := json.Marshal([]any{tool})
	if err != nil {
		t.Fatal(err)
	}

	result, err := anthropicToolsWireToOpenAI(toolsJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	fn, _ := result[0]["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)

	// $schema should be stripped
	if _, ok := params["$schema"]; ok {
		t.Error("$schema should be stripped from parameters")
	}

	// propertyNames should be stripped from answers
	answers, _ := params["properties"].(map[string]any)["answers"].(map[string]any)
	if _, ok := answers["propertyNames"]; ok {
		t.Error("propertyNames should be stripped from answers")
	}

	// propertyNames should be stripped from annotations
	annotations, _ := params["properties"].(map[string]any)["annotations"].(map[string]any)
	if _, ok := annotations["propertyNames"]; ok {
		t.Error("propertyNames should be stripped from annotations")
	}
}

func TestAnthropicToolsWireToOpenAI_EmptyTools(t *testing.T) {
	result, err := anthropicToolsWireToOpenAI(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("nil input should return nil")
	}

	result, err = anthropicToolsWireToOpenAI(json.RawMessage("null"))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("null input should return nil")
	}
}

func TestAnthropicToolsWireToOpenAI_FiltersAdvisorAndComputer(t *testing.T) {
	tools := []map[string]any{
		{"name": "normal_tool", "input_schema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "advisor", "type": "advisor_20260301"},
		{"name": "computer", "type": "computer_20250124"},
		{"name": "server_tool", "type": "server"},
	}
	toolsJSON, _ := json.Marshal(tools)
	result, err := anthropicToolsWireToOpenAI(toolsJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 tool after filtering, got %d", len(result))
	}
	fn := result[0]["function"].(map[string]any)
	if fn["name"] != "normal_tool" {
		t.Errorf("expected normal_tool, got %v", fn["name"])
	}
}
