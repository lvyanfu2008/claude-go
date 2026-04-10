package toolpool

import (
	"goc/commands"
	"goc/types"
)

// ToolSpecsFromEmbeddedToolsAPIJSON parses commands.ToolsAPIJSON (data/tools_api.json).
func ToolSpecsFromEmbeddedToolsAPIJSON() ([]types.ToolSpec, error) {
	return ParseToolsAPIDocumentJSON(commands.ToolsAPIJSON)
}
