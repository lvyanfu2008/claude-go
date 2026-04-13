package bashzog

import (
	"encoding/json"
	"fmt"
)

// bashZogAPIExport is the Messages API shape for the BashZog tool row (snake_case keys).
type bashZogAPIExport struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ExportBashZogToolJSON returns pretty-printed JSON for the BashZog tool row: same description
// and input_schema as the embedded bash snapshot, with name forced to [ZogToolName].
func ExportBashZogToolJSON() ([]byte, error) {
	d, err := LoadAPIData()
	if err != nil {
		return nil, err
	}
	out := bashZogAPIExport{
		Name:        ZogToolName,
		Description: d.Description,
		InputSchema: append(json.RawMessage(nil), d.InputSchemaRaw...),
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("bashzog export: %w", err)
	}
	return b, nil
}
