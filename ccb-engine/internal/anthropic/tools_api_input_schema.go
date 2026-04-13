package anthropic

import (
	"encoding/json"
	"sync"

	"goc/commands"
)

var (
	exportParseOnce sync.Once
	exportSchemas   map[string]json.RawMessage
	exportParseErr  error
)

func loadToolsAPIExportSchemas() {
	exportSchemas = make(map[string]json.RawMessage)
	var doc struct {
		Tools []struct {
			Name        string          `json:"name"`
			InputSchema json.RawMessage `json:"input_schema"`
		} `json:"tools"`
	}
	raw := commands.ToolsAPIJSON
	if len(raw) == 0 {
		exportParseErr = errToolsAPIEmpty{}
		return
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		exportParseErr = err
		return
	}
	for _, row := range doc.Tools {
		n := row.Name
		if n == "" {
			continue
		}
		if len(row.InputSchema) > 0 {
			exportSchemas[n] = row.InputSchema
		}
	}
}

type errToolsAPIEmpty struct{}

func (errToolsAPIEmpty) Error() string {
	return "commands.ToolsAPIJSON is empty"
}

// InputSchemaFromTSAPIExport returns input_schema from embedded commands/data/tools_api.json
// (claude-code scripts/export-tools-registry-json.ts → toolToAPISchema). The bool is false
// when the tool is absent from the export or embed failed to parse.
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

// parityInputSchema uses the TS export when present; otherwise returns fallback (for tools
// not included in a given export snapshot, e.g. MCP helpers, SendMessage).
func parityInputSchema(exportToolName string, fallback any) any {
	if s, ok := InputSchemaFromTSAPIExport(exportToolName); ok {
		return s
	}
	return fallback
}

// mustExportInputSchema returns input_schema from embedded tools_api.json or panics.
// Use for built-ins that must stay locked to TS toolToAPISchema output.
func mustExportInputSchema(toolName string) any {
	s, ok := InputSchemaFromTSAPIExport(toolName)
	if !ok {
		panic("anthropic: missing input_schema for tool " + toolName + " in commands.ToolsAPIJSON (sync from claude-code export:tools-registry)")
	}
	return s
}
