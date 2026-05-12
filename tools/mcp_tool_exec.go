package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// MCPToolExecutor executes MCP tools discovered from connected servers.
type MCPToolExecutor struct {
	// ExecuteMCPToolFunc is the function that actually calls an MCP tool.
	// In production, this delegates to the mcp.ConnectionManager.
	ExecuteMCPToolFunc func(ctx context.Context, fullToolName string, args map[string]interface{}) (string, error)
}

// DefaultMCPToolExecutor is the global executor instance. Set this during
// app initialization when the MCP connection manager is available.
var DefaultMCPToolExecutor = &MCPToolExecutor{}

// ExecuteMCPTool executes an MCP tool by its full mcp__<server>__<tool> name.
func (e *MCPToolExecutor) ExecuteMCPTool(ctx context.Context, fullToolName string, argsJSON json.RawMessage) (string, error) {
	if e.ExecuteMCPToolFunc == nil {
		return "", fmt.Errorf("MCP tool executor not configured")
	}

	var args map[string]interface{}
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("parse MCP tool args: %w", err)
		}
	}

	return e.ExecuteMCPToolFunc(ctx, fullToolName, args)
}

// ListMcpResources lists resources from an MCP server.
func (e *MCPToolExecutor) ListMcpResources(ctx context.Context, server string) (string, error) {
	if e.ExecuteMCPToolFunc == nil {
		return "", fmt.Errorf("MCP tool executor not configured")
	}

	// ListMcpResourcesTool is a special internal tool.
	// The server param is forwarded to the connection manager.
	result, err := e.ExecuteMCPToolFunc(ctx, fmt.Sprintf("__list_resources__%s", server), map[string]interface{}{
		"server": server,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// ReadMcpResource reads a specific resource from an MCP server.
func (e *MCPToolExecutor) ReadMcpResource(ctx context.Context, server, uri string) (string, error) {
	if e.ExecuteMCPToolFunc == nil {
		return "", fmt.Errorf("MCP tool executor not configured")
	}

	result, err := e.ExecuteMCPToolFunc(ctx, fmt.Sprintf("__read_resource__%s", server), map[string]interface{}{
		"server": server,
		"uri":    uri,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// IsMcpTool checks if the given tool name is an MCP tool.
func IsMcpTool(toolName string) bool {
	return len(toolName) > 5 && toolName[:5] == "mcp__"
}

// RunMcpTool executes an MCP tool and returns the result as a string.
// This is the main entry point called by the tool execution engine.
func RunMcpTool(ctx context.Context, toolName string, input json.RawMessage) (string, error) {
	log.Printf("[mcp-exec] executing MCP tool: %s", toolName)
	result, err := DefaultMCPToolExecutor.ExecuteMCPTool(ctx, toolName, input)
	if err != nil {
		log.Printf("[mcp-exec] error executing %s: %v", toolName, err)
		return "", err
	}
	log.Printf("[mcp-exec] result from %s: %d chars", toolName, len(result))
	return result, nil
}
