package anthropic

import (
	"encoding/json"
	"sync"

	"goc/tools/toolpool"
)

var (
	exportParseOnce sync.Once
	exportSchemas   map[string]json.RawMessage
	exportParseErr  error
)

func loadToolsAPIExportSchemas() {
	exportSchemas = make(map[string]json.RawMessage)
	for _, spec := range toolpool.ToolSpecsFromGoWire() {
		n := spec.Name
		if n == "" {
			continue
		}
		if len(spec.InputJSONSchema) > 0 {
			exportSchemas[n] = append(json.RawMessage(nil), spec.InputJSONSchema...)
		}
	}
}

// InputSchemaFromTSAPIExport returns input_schema from the Go tool wire.
// The bool is false when the tool is absent from the Go runtime registry.
func InputSchemaFromTSAPIExport(toolName string) (any, bool) {
	exportParseOnce.Do(loadToolsAPIExportSchemas)
	if exportParseErr != nil {
		return nil, false
	}
	raw, ok := exportSchemas[toolName]
	if !ok || len(raw) == 0 {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	return v, true
}

// parityInputSchema uses the Go tool wire when present; otherwise returns fallback
// (for tools not included in the runtime registry, e.g. MCP helpers).
func parityInputSchema(exportToolName string, fallback any) any {
	if s, ok := InputSchemaFromTSAPIExport(exportToolName); ok {
		return s
	}
	return fallback
}

// mustExportInputSchema returns input_schema from the Go tool wire or panics.
// Use for built-ins that must stay locked to the runtime tool registry output.
func mustExportInputSchema(toolName string) any {
	s, ok := InputSchemaFromTSAPIExport(toolName)
	if !ok {
		panic("anthropic: missing input_schema for tool " + toolName + " in go tool wire")
	}
	return s
}
