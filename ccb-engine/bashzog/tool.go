package bashzog

import (
	"encoding/json"
	"sync"

	"goc/types"
)

// ZogToolName is the tool_use name for the Go Zog-validated Bash sibling (same execution as "Bash").
const ZogToolName = "BashZog"

type bashWire struct {
	Name           string
	Description    string
	InputSchemaRaw json.RawMessage
	inputSchemaObj map[string]any
}

var (
	wireOnce sync.Once
	wireErr  error
	wire     bashWire
)

func loadWire() {
	schemaMap := bashToolInputSchema()
	schemaBytes, err := json.Marshal(schemaMap)
	if err != nil {
		wireErr = err
		return
	}
	wire = bashWire{
		Name:           bashModelWireName,
		Description:    GetSimplePrompt(),
		InputSchemaRaw: append(json.RawMessage(nil), schemaBytes...),
		inputSchemaObj: schemaMap,
	}
}

// APIData is the Messages API Bash row when GO_TOOL_INPUT_VALIDATOR=zog (no runtime tools_api read).
type APIData struct {
	Name           string
	Description    string
	InputSchema    map[string]any
	InputSchemaRaw json.RawMessage
}

// LoadAPIData returns the Go-sourced Bash model snapshot (name, description, input_schema).
func LoadAPIData() (APIData, error) {
	wireOnce.Do(loadWire)
	if wireErr != nil {
		return APIData{}, wireErr
	}
	return APIData{
		Name:           wire.Name,
		Description:    wire.Description,
		InputSchema:    wire.inputSchemaObj,
		InputSchemaRaw: append(json.RawMessage(nil), wire.InputSchemaRaw...),
	}, nil
}

// BashToolSpec returns a [types.ToolSpec] using [bashModelWireName], [GetSimplePrompt], and [bashToolInputSchema].
// Prefer [BashZogToolSpec] when wiring the Zog-specific tool row.
func BashToolSpec() (types.ToolSpec, error) {
	d, err := LoadAPIData()
	if err != nil {
		return types.ToolSpec{}, err
	}
	return types.ToolSpec{
		Name:            d.Name,
		Description:     d.Description,
		InputJSONSchema: append(json.RawMessage(nil), d.InputSchemaRaw...),
	}, nil
}

// BashZogToolSpec returns a [types.ToolSpec] for [ZogToolName] using the same description and schema as [BashToolSpec].
func BashZogToolSpec() (types.ToolSpec, error) {
	d, err := LoadAPIData()
	if err != nil {
		return types.ToolSpec{}, err
	}
	return types.ToolSpec{
		Name:            ZogToolName,
		Description:     d.Description,
		InputJSONSchema: append(json.RawMessage(nil), d.InputSchemaRaw...),
	}, nil
}
