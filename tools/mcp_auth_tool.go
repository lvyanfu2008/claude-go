package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/types"
)

// McpAuthToolHandler is called by McpAuthFromJSON to execute the OAuth flow.
// It receives the server name parsed from the full tool name (mcp__<server>__authenticate).
// It returns a JSON-marshalable result with status and either auth_url or error.
// Set by the MCP lifecycle manager during bootstrap via SetMcpAuthToolHandler.
var McpAuthToolHandler func(ctx context.Context, serverName string) (map[string]any, error)

// McpAuthCallResult mirrors the TS McpAuthOutput shape.
type McpAuthCallResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	AuthURL string `json:"authUrl,omitempty"`
}

// SetMcpAuthToolHandler sets the global MCP auth tool handler.
func SetMcpAuthToolHandler(handler func(ctx context.Context, serverName string) (map[string]any, error)) {
	McpAuthToolHandler = handler
}

// BuildMcpAuthToolName builds the pseudo-tool name for an MCP server's auth tool.
// Mirrors TS buildMcpToolName(serverName, 'authenticate').
func BuildMcpAuthToolName(serverName string) string {
	return fmt.Sprintf("mcp__%s__authenticate", serverName)
}

// CreateMcpAuthToolSpec creates a ToolSpec for the MCP auth pseudo-tool.
func CreateMcpAuthToolSpec(serverName, description string) types.ToolSpec {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	return types.ToolSpec{
		Name:            BuildMcpAuthToolName(serverName),
		Description:     description,
		InputJSONSchema: json.RawMessage(mustMarshalJSONRaw(schema)),
		IsMcp:           boolPtr(true),
		MCPInfo: &types.MCPInfo{
			ServerName: serverName,
			ToolName:   "authenticate",
		},
	}
}

// McpAuthFromJSON handles the mcp__<server>__authenticate tool call.
// It parses the server name from the tool name and delegates to McpAuthToolHandler.
func McpAuthFromJSON(ctx context.Context, toolName string, raw []byte) (string, bool, error) {
	serverName := parseServerFromMcpAuthName(toolName)
	if serverName == "" {
		out := map[string]any{
			"data": McpAuthCallResult{
				Status:  "error",
				Message: fmt.Sprintf("Could not parse server name from tool: %s", toolName),
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	if McpAuthToolHandler == nil {
		out := map[string]any{
			"data": McpAuthCallResult{
				Status:  "error",
				Message: "MCP OAuth handler is not configured.",
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	result, err := McpAuthToolHandler(ctx, serverName)
	if err != nil {
		out := map[string]any{
			"data": McpAuthCallResult{
				Status:  "error",
				Message: err.Error(),
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	out := map[string]any{"data": result}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// parseServerFromMcpAuthName extracts the server name from mcp__<server>__authenticate.
func parseServerFromMcpAuthName(toolName string) string {
	const prefix = "mcp__"
	const suffix = "__authenticate"
	if len(toolName) > len(prefix)+len(suffix) &&
		toolName[:len(prefix)] == prefix &&
		toolName[len(toolName)-len(suffix):] == suffix {
		return toolName[len(prefix) : len(toolName)-len(suffix)]
	}
	return ""
}

func mustMarshalJSONRaw(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("McpAuthTool: failed to marshal JSON: %v", err))
	}
	return b
}

func boolPtr(b bool) *bool {
	return &b
}
