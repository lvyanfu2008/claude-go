package toolpool

import (
	"strings"

	"goc/types"
)

// IsMcpTool mirrors isMcpTool in src/services/mcp/utils.ts (lines 245–247).
func IsMcpTool(tool types.ToolSpec) bool {
	if tool.Name != "" && strings.HasPrefix(tool.Name, "mcp__") {
		return true
	}
	if tool.IsMcp != nil && *tool.IsMcp {
		return true
	}
	return false
}
