package toolpool

import (
	"encoding/json"

	"goc/types"
)

// MarshalToolsAPIDocumentDefinitions encodes tools for ToolUseContext.Options.Tools (Anthropic-style array).
func MarshalToolsAPIDocumentDefinitions(tools []types.ToolSpec) (json.RawMessage, error) {
	return MarshalToolsAPIDocumentDefinitionsWithOptions(tools, DefaultToolToAPISchemaOptionsFromEnv())
}

// MarshalToolsAPIDocumentDefinitionsWithOptions encodes tools for ToolUseContext.Options.Tools
// with per-request tool schema overlay behavior mirroring TS toolToAPISchema.
func MarshalToolsAPIDocumentDefinitionsWithOptions(tools []types.ToolSpec, opts ToolToAPISchemaOptions) (json.RawMessage, error) {
	defs := make([]APIToolDefinition, 0, len(tools))
	for _, t := range tools {
		deferTool := opts
		// TS defers per-call via options.deferLoading. For current Go callers we
		// preserve legacy behavior by honoring ToolSpec.ShouldDefer as default.
		if !opts.DeferLoading && t.ShouldDefer != nil {
			deferTool.DeferLoading = *t.ShouldDefer
		}
		defs = append(defs, ToolToAPISchema(t, deferTool))
	}
	return json.Marshal(defs)
}
