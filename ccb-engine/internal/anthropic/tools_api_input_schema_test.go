package anthropic

import (
	"encoding/json"
	"reflect"
	"testing"

	"goc/commands"
)

// TestParityToolInputSchemasMatchToolsAPIExport ensures GouParityToolList uses the same
// input_schema objects as commands.ToolsAPIJSON for every tool name present in the export.
func TestParityToolInputSchemasMatchToolsAPIExport(t *testing.T) {
	var doc struct {
		Tools []struct {
			Name        string          `json:"name"`
			InputSchema json.RawMessage `json:"input_schema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(commands.ToolsAPIJSON, &doc); err != nil {
		t.Fatal(err)
	}
	exportByName := make(map[string]json.RawMessage, len(doc.Tools))
	for _, row := range doc.Tools {
		if row.Name != "" && len(row.InputSchema) > 0 {
			exportByName[row.Name] = row.InputSchema
		}
	}
	for _, def := range GouParityToolList() {
		want, ok := exportByName[def.Name]
		if !ok {
			continue
		}
		gotBytes, err := json.Marshal(def.InputSchema)
		if err != nil {
			t.Fatalf("%s: marshal InputSchema: %v", def.Name, err)
		}
		var gotObj, wantObj any
		if err := json.Unmarshal(gotBytes, &gotObj); err != nil {
			t.Fatalf("%s: unmarshal got: %v", def.Name, err)
		}
		if err := json.Unmarshal(want, &wantObj); err != nil {
			t.Fatalf("%s: unmarshal export: %v", def.Name, err)
		}
		if !reflect.DeepEqual(gotObj, wantObj) {
			t.Fatalf("%s: InputSchema differs from tools_api.json export (sync via export:tools-registry + copy to commands/data)", def.Name)
		}
	}
}
