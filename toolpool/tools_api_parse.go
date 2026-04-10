package toolpool

import (
	"encoding/json"
	"fmt"

	"goc/types"
)

// toolsAPIDocument mirrors the export JSON from scripts/export-tools-registry-json.ts.
type toolsAPIDocument struct {
	Tools []toolsAPIEntry `json:"tools"`
}

type toolsAPIEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ParseToolsAPIDocumentJSON parses export JSON (meta + tools[]) into ToolSpec slices.
// Each entry maps input_schema → ToolSpec.InputJSONSchema (src/types mirror).
func ParseToolsAPIDocumentJSON(data []byte) ([]types.ToolSpec, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("toolpool: empty tools API JSON")
	}
	var doc toolsAPIDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make([]types.ToolSpec, 0, len(doc.Tools))
	for _, e := range doc.Tools {
		if e.Name == "" {
			continue
		}
		out = append(out, types.ToolSpec{
			Name:               e.Name,
			Description:        e.Description,
			InputJSONSchema:    e.InputSchema,
			MaxResultSizeChars: 0,
		})
	}
	return out, nil
}
