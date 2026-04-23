package bashzog

import (
	"encoding/json"
	"strings"
	"sync"

	"goc/commands/featuregates"
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
		Description:    bashToolModelDescription,
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

// addMonitorToolDescriptionToBashPrompt modifies the Bash tool description to include Monitor tool information
// when MONITOR_TOOL feature is enabled, mirroring TypeScript behavior from prompt.ts
func addMonitorToolDescriptionToBashPrompt(description string) string {
	if !featuregates.Feature("MONITOR_TOOL") {
		return description
	}

	// Find the position where we need to insert Monitor tool information
	// Looking for the sleep section that starts with "Do not sleep between commands..."
	sleepSectionStart := "  - Do not sleep between commands that can run immediately — just run them."

	if !strings.Contains(description, sleepSectionStart) {
		// If we can't find the expected structure, return as-is
		return description
	}

	// Add Monitor tool description after the first sleep bullet point
	monitorInstruction := "\n  - Use the Monitor tool to stream events from a background process (each stdout line is a notification). For one-shot \"wait until done,\" use Bash with run_in_background instead."

	// Replace the first sleep instruction with itself plus the monitor instruction
	modifiedDescription := strings.Replace(
		description,
		sleepSectionStart,
		sleepSectionStart+monitorInstruction,
		1,
	)

	return modifiedDescription
}

// BashToolSpec returns a [types.ToolSpec] using [bashModelWireName], [bashToolModelDescription], and [bashToolInputSchema].
// Prefer [BashZogToolSpec] when wiring the Zog-specific tool row.
func BashToolSpec() (types.ToolSpec, error) {
	d, err := LoadAPIData()
	if err != nil {
		return types.ToolSpec{}, err
	}
	// Apply Monitor tool description modifications if feature is enabled
	description := addMonitorToolDescriptionToBashPrompt(d.Description)

	return types.ToolSpec{
		Name:            d.Name,
		Description:     description,
		InputJSONSchema: append(json.RawMessage(nil), d.InputSchemaRaw...),
	}, nil
}

// BashZogToolSpec returns a [types.ToolSpec] for [ZogToolName] using the same description and schema as [BashToolSpec].
func BashZogToolSpec() (types.ToolSpec, error) {
	d, err := LoadAPIData()
	if err != nil {
		return types.ToolSpec{}, err
	}
	// Apply Monitor tool description modifications if feature is enabled
	description := addMonitorToolDescriptionToBashPrompt(d.Description)

	return types.ToolSpec{
		Name:            ZogToolName,
		Description:     description,
		InputJSONSchema: append(json.RawMessage(nil), d.InputSchemaRaw...),
	}, nil
}
