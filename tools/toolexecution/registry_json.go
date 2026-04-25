package toolexecution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/internal/jsonschemavalidate"
	"goc/internal/toolvalidator"
	"goc/types"
)

// jsonToolRegistry implements [ToolRegistry] from API-shaped tools[] JSON (name + input_schema / inputSchema).
type jsonToolRegistry struct {
	byName map[string]*jsonSchemaTool
}

// NewJSONToolRegistry parses tools JSON (array of objects with name and input_schema) into a registry.
// Unknown or invalid entries are skipped; empty array yields empty registry (Find always false).
func NewJSONToolRegistry(toolsJSON json.RawMessage) (ToolRegistry, error) {
	r := &jsonToolRegistry{byName: make(map[string]*jsonSchemaTool)}
	if len(bytesTrim(toolsJSON)) == 0 {
		return r, nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(toolsJSON, &arr); err != nil {
		return nil, fmt.Errorf("toolexecution: tools json: %w", err)
	}
	var primaries []*jsonSchemaTool
	for _, o := range arr {
		n, _ := o["name"].(string)
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		schema := pickInputSchema(o)
		t := &jsonSchemaTool{name: n, schema: schema, aliases: stringSlice(o["aliases"])}
		r.byName[n] = t
		primaries = append(primaries, t)
	}
	for _, t := range primaries {
		for _, a := range t.aliases {
			a = strings.TrimSpace(a)
			if a == "" || a == t.name {
				continue
			}
			if _, exists := r.byName[a]; !exists {
				r.byName[a] = t
			}
		}
	}
	return r, nil
}

func bytesTrim(b json.RawMessage) []byte { return []byte(strings.TrimSpace(string(b))) }

func pickInputSchema(o map[string]any) any {
	if v, ok := o["input_schema"]; ok {
		return v
	}
	if v, ok := o["inputSchema"]; ok {
		return v
	}
	return nil
}

func stringSlice(v any) []string {
	raw, _ := json.Marshal(v)
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

type jsonSchemaTool struct {
	name    string
	aliases []string
	schema  any
}

func (t *jsonSchemaTool) Name() string { return t.name }

func (t *jsonSchemaTool) Aliases() []string { return t.aliases }

// InputSchemaAny returns the parsed input_schema / inputSchema blob for early validation ([CheckPermissionsAndCallTool]).
func (t *jsonSchemaTool) InputSchemaAny() any { return t.schema }

func (t *jsonSchemaTool) Call(
	ctx context.Context,
	toolUseID string,
	input json.RawMessage,
	tcx *ToolUseContext,
	canUseTool CanUseToolFn,
	assistant AssistantMeta,
	onProgress func(toolUseID string, data json.RawMessage),
) (*types.ToolRunResult, error) {
	_ = assistant
	_ = onProgress
	if canUseTool != nil && tcx != nil {
		if err := canUseTool(t.name, input, tcx); err != nil {
			return nil, err
		}
	}
	if !(toolvalidator.InputValidatorMode() == "zog" && strings.EqualFold(t.name, bashzog.ZogToolName)) {
		if err := ValidateInputAgainstSchema(t.name, t.schema, input); err != nil {
			return nil, fmt.Errorf("InputValidationError: %s", jsonschemavalidate.FormatInputValidationError(t.name, err))
		}
	}
	data := append(json.RawMessage(nil), input...)
	if len(bytesTrim(data)) == 0 {
		data = json.RawMessage(`{}`)
	}
	return &types.ToolRunResult{Data: data}, nil
}

func (r *jsonToolRegistry) FindToolByName(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.byName[strings.TrimSpace(name)]
	return t, ok
}
