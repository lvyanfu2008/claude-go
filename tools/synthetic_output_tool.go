package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const SyntheticOutputToolName = "StructuredOutput"

type SyntheticOutputEnabledOpts struct {
	IsNonInteractiveSession bool
}

func IsSyntheticOutputToolEnabled(opts SyntheticOutputEnabledOpts) bool {
	return opts.IsNonInteractiveSession
}

type PermissionResult struct {
	Behavior   string         `json:"behavior"`
	UpdatedRaw map[string]any `json:"updatedInput,omitempty"`
}

type ToolCallResult struct {
	Data             string         `json:"data"`
	StructuredOutput map[string]any `json:"structured_output"`
}

type SyntheticOutputTool struct {
	InputJSONSchema map[string]any
	validate        *jsonschema.Schema
}

func (t *SyntheticOutputTool) Name() string { return SyntheticOutputToolName }

func (t *SyntheticOutputTool) Description() string {
	return "Return structured output in the requested format"
}

func (t *SyntheticOutputTool) Prompt() string {
	return "Use this tool to return your final response in the requested structured format. You MUST call this tool exactly once at the end of your response to provide the structured output."
}

func (t *SyntheticOutputTool) Call(input map[string]any) (ToolCallResult, error) {
	if t.validate != nil {
		ins, err := marshalAsJSONValue(input)
		if err != nil {
			return ToolCallResult{}, err
		}
		if err := t.validate.Validate(ins); err != nil {
			detail := strings.TrimSpace(err.Error())
			return ToolCallResult{}, fmt.Errorf("Output does not match required schema: %s", detail)
		}
	}
	return ToolCallResult{
		Data:             "Structured output provided successfully",
		StructuredOutput: input,
	}, nil
}

func (t *SyntheticOutputTool) CheckPermissions(input map[string]any) PermissionResult {
	return PermissionResult{
		Behavior:   "allow",
		UpdatedRaw: input,
	}
}

func (t *SyntheticOutputTool) RenderToolUseMessage(input map[string]any) *string {
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return nil
	}
	if len(keys) <= 3 {
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, jsonStringify(input[k])))
		}
		s := strings.Join(parts, ", ")
		return &s
	}
	s := fmt.Sprintf("%d fields: %s…", len(keys), strings.Join(keys[:3], ", "))
	return &s
}

func (t *SyntheticOutputTool) RenderToolUseRejectedMessage() string {
	return "Structured output rejected"
}
func (t *SyntheticOutputTool) RenderToolUseErrorMessage() string     { return "Structured output error" }
func (t *SyntheticOutputTool) RenderToolUseProgressMessage() *string { return nil }
func (t *SyntheticOutputTool) RenderToolResultMessage(output string) string {
	return output
}

type ToolResultBlockParam struct {
	ToolUseID string `json:"tool_use_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
}

func (t *SyntheticOutputTool) MapToolResultToToolResultBlockParam(content, toolUseID string) ToolResultBlockParam {
	return ToolResultBlockParam{
		ToolUseID: toolUseID,
		Type:      "tool_result",
		Content:   content,
	}
}

type CreateResult struct {
	Tool  *SyntheticOutputTool
	Error string
}

var syntheticOutputToolCache sync.Map // map[uintptr]CreateResult (keyed by map identity)

func CreateSyntheticOutputTool(jsonSchema map[string]any) CreateResult {
	ptr := reflect.ValueOf(jsonSchema).Pointer()
	if cached, ok := syntheticOutputToolCache.Load(ptr); ok {
		return cached.(CreateResult)
	}
	result := buildSyntheticOutputTool(jsonSchema)
	syntheticOutputToolCache.Store(ptr, result)
	return result
}

func buildSyntheticOutputTool(jsonSchema map[string]any) CreateResult {
	schemaObj, err := compileSchema(jsonSchema)
	if err != nil {
		return CreateResult{Error: err.Error()}
	}
	return CreateResult{
		Tool: &SyntheticOutputTool{
			InputJSONSchema: jsonSchema,
			validate:        schemaObj,
		},
	}
}

func compileSchema(schema map[string]any) (*jsonschema.Schema, error) {
	loc := "https://goc.local/tools/synthetic-output"
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	doc, err := marshalAsJSONValue(schema)
	if err != nil {
		return nil, err
	}
	if err := c.AddResource(loc, doc); err != nil {
		return nil, err
	}
	return c.Compile(loc)
}

func marshalAsJSONValue(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return dec, nil
}

func jsonStringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}
