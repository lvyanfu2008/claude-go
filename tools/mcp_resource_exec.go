package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPResourceReader reads resources from MCP servers.
var MCPResourceReader interface {
	ListResources(ctx context.Context, server string) ([]ResourceItem, error)
	ReadResource(ctx context.Context, server, uri string) (string, error)
}

// ResourceItem is a simplified resource description.
type ResourceItem struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Server      string `json:"server"`
}

// ListMcpResourcesHandler handles ListMcpResourcesTool invocations.
// Returns a JSON array of resource descriptions for the given server (or all servers).
func ListMcpResourcesHandler(ctx context.Context, input json.RawMessage) (string, error) {
	if MCPResourceReader == nil {
		return "", fmt.Errorf("MCP resource reader not configured")
	}

	var params struct {
		Server string `json:"server"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return "", fmt.Errorf("parse ListMcpResources params: %w", err)
		}
	}

	resources, err := MCPResourceReader.ListResources(ctx, params.Server)
	if err != nil {
		return "", fmt.Errorf("list MCP resources: %w", err)
	}

	output, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal resources: %w", err)
	}

	return string(output), nil
}

// ReadMcpResourceHandler handles ReadMcpResourceTool invocations.
func ReadMcpResourceHandler(ctx context.Context, input json.RawMessage) (string, error) {
	if MCPResourceReader == nil {
		return "", fmt.Errorf("MCP resource reader not configured")
	}

	var params struct {
		Server string `json:"server"`
		URI    string `json:"uri"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("parse ReadMcpResource params: %w", err)
	}

	if params.Server == "" || params.URI == "" {
		return "", fmt.Errorf("both 'server' and 'uri' are required")
	}

	result, err := MCPResourceReader.ReadResource(ctx, params.Server, params.URI)
	if err != nil {
		return "", fmt.Errorf("read MCP resource %q from %q: %w", params.URI, params.Server, err)
	}

	return result, nil
}
