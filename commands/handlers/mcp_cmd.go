package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// McpListResult is the JSON payload for /mcp.
type McpListResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleMcpCommand handles /mcp — lists configured MCP servers.
func HandleMcpCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()

	var lines []string
	lines = append(lines, "MCP Server Configuration:")

	// Check .mcp.json
	mcpJSONPath := filepath.Join(cwd, ".mcp.json")
	if data, err := os.ReadFile(mcpJSONPath); err == nil {
		lines = append(lines, fmt.Sprintf("\n.mcp.json: %s (%d bytes)", mcpJSONPath, len(data)))
	}

	// Check settings.go.json
	settingsGoPath := filepath.Join(cwd, ".harness", "settings.go.json")
	if data, err := os.ReadFile(settingsGoPath); err == nil {
		lines = append(lines, fmt.Sprintf("settings.go.json: %s (%d bytes)", settingsGoPath, len(data)))
	}

	lines = append(lines, "\nUse `claude mcp list` for a detailed list.")
	lines = append(lines, "Use `claude mcp add [name] [type] [url]` to add a server.")
	return json.Marshal(McpListResult{
		Type: "text", Value: strings.Join(lines, "\n"),
	})
}
