// MCP tools JSON (API-shaped definitions) for assembleToolPool without an MCP host.
// Format: JSON array of { "name": "mcp__server__tool", ... } or { "serverName", "toolName", "input_schema", ... }.
// Env: GOU_DEMO_MCP_TOOLS_JSON — path (same resolution as GOU_DEMO_MCP_COMMANDS_JSON).
package mcpcommands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/permissionrules"
	"goc/types"
)

// EnvToolsJSONPath is the environment variable for MCP **tool definitions** (not slash commands).
const EnvToolsJSONPath = "GOU_DEMO_MCP_TOOLS_JSON"

type mcpToolFileEntry struct {
	Name        string          `json:"name"`
	ServerName  string          `json:"serverName"`
	ToolName    string          `json:"toolName"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// LoadToolsFromPath reads a JSON array of MCP tool definitions into []types.ToolSpec (IsMcp set, MCPInfo when parseable).
// Empty path returns (nil, nil). Mirrors TS appState.mcp.tools entries passed to assembleToolPool.
func LoadToolsFromPath(path string) ([]types.ToolSpec, error) {
	resolved, err := resolveMCPCommandsPath(path)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		return nil, nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("mcpcommands: read tools %q: %w", resolved, err)
	}
	return ParseMCPToolsJSON(data)
}

// LoadToolsFromEnv loads from os.Getenv(EnvToolsJSONPath).
func LoadToolsFromEnv() ([]types.ToolSpec, error) {
	return LoadToolsFromPath(os.Getenv(EnvToolsJSONPath))
}

// ParseMCPToolsJSON parses raw JSON (exported for tests).
func ParseMCPToolsJSON(data []byte) ([]types.ToolSpec, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var entries []mcpToolFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("mcpcommands: parse mcp tools: %w", err)
	}
	out := make([]types.ToolSpec, 0, len(entries))
	for _, e := range entries {
		spec, err := toolSpecFromMCPFileEntry(e)
		if err != nil {
			return nil, err
		}
		if spec.Name != "" {
			out = append(out, spec)
		}
	}
	return out, nil
}

func toolSpecFromMCPFileEntry(e mcpToolFileEntry) (types.ToolSpec, error) {
	t := true
	spec := types.ToolSpec{
		Description:        e.Description,
		InputJSONSchema:    e.InputSchema,
		MaxResultSizeChars: 0,
		IsMcp:              &t,
	}
	switch {
	case e.Name != "":
		if !strings.HasPrefix(e.Name, "mcp__") {
			return types.ToolSpec{}, fmt.Errorf("mcpcommands: tool %q must start with mcp__ or use serverName+toolName", e.Name)
		}
		spec.Name = e.Name
		if info := permissionrules.McpInfoFromString(e.Name); info != nil && info.ToolName != nil && *info.ToolName != "" {
			spec.MCPInfo = &types.MCPInfo{
				ServerName: info.ServerName,
				ToolName:   *info.ToolName,
			}
		}
	case e.ServerName != "" && e.ToolName != "":
		spec.Name = permissionrules.BuildMcpToolName(e.ServerName, e.ToolName)
		spec.MCPInfo = &types.MCPInfo{ServerName: e.ServerName, ToolName: e.ToolName}
	default:
		return types.ToolSpec{}, nil
	}
	return spec, nil
}
