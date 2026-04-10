package permissionrules

import (
	"strings"

	"goc/types"
)

// McpInfoFromString mirrors mcpInfoFromString in src/services/mcp/mcpStringUtils.ts.
func McpInfoFromString(toolString string) *McpInfoStringResult {
	parts := strings.Split(toolString, "__")
	if len(parts) < 2 {
		return nil
	}
	mcpPart := parts[0]
	serverName := parts[1]
	if mcpPart != "mcp" || serverName == "" {
		return nil
	}
	var toolName *string
	if len(parts) > 2 {
		joined := strings.Join(parts[2:], "__")
		toolName = &joined
	}
	return &McpInfoStringResult{ServerName: serverName, ToolName: toolName}
}

// McpInfoStringResult holds parsed MCP server/tool from an mcp__ qualified name.
type McpInfoStringResult struct {
	ServerName string
	ToolName   *string // nil when server-only (e.g. mcp__server)
}

// GetMcpPrefix mirrors getMcpPrefix in src/services/mcp/mcpStringUtils.ts.
func GetMcpPrefix(serverName string) string {
	return "mcp__" + NormalizeNameForMCP(serverName) + "__"
}

// BuildMcpToolName mirrors buildMcpToolName in src/services/mcp/mcpStringUtils.ts.
func BuildMcpToolName(serverName, toolName string) string {
	return GetMcpPrefix(serverName) + NormalizeNameForMCP(toolName)
}

// GetToolNameForPermissionCheck mirrors getToolNameForPermissionCheck in src/services/mcp/mcpStringUtils.ts.
func GetToolNameForPermissionCheck(tool types.ToolSpec) string {
	if tool.MCPInfo != nil {
		return BuildMcpToolName(tool.MCPInfo.ServerName, tool.MCPInfo.ToolName)
	}
	return tool.Name
}
