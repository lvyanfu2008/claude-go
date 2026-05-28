package toolsearch

import (
	"strings"

	"goc/internal/anthropic"
)

const (
	// ToolSearchToolName matches src/tools/ToolSearchTool/constants.ts
	ToolSearchToolName = "ToolSearch"
)

// IsDeferredTool mirrors TS isDeferredTool. A tool is deferred when its ToolSpec.ShouldDefer
// was set (serialized as DeferLoading in the API schema) or when it's an MCP tool (mcp__ prefix).
func IsDeferredTool(t anthropic.ToolDefinition) bool {
	if t.Name == ToolSearchToolName {
		return false
	}
	if t.DeferLoading != nil && *t.DeferLoading {
		return true
	}
	if strings.HasPrefix(t.Name, "mcp__") {
		return true
	}
	return false
}

// findDeferredByName looks up a tool name in the tools slice and checks if it is deferred.
// Returns false when the name is not found.
func findDeferredByName(tools []anthropic.ToolDefinition, name string) bool {
	for i := range tools {
		if tools[i].Name == name {
			return IsDeferredTool(tools[i])
		}
	}
	return false
}
