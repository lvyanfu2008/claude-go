package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// DiscoveryResult holds the discovered tools, commands, and resources from an MCP server.
type DiscoveryResult struct {
	Tools     []SerializedTool
	Commands  []MCPServerCommand
	Resources []ServerResource
}

// MCPServerCommand is a prompt/command discovered from an MCP server.
// MCP "prompts" are mapped to Claude Code "commands".
type MCPServerCommand struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ServerName  string `json:"serverName"`
}

// FetchToolsForClient fetches all tools from a connected MCP client.
// Mirrors TS fetchToolsForClient.
func FetchToolsForClient(ctx context.Context, c *client.Client, serverName string) ([]SerializedTool, error) {
	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list tools from %q: %w", serverName, err)
	}

	tools := make([]SerializedTool, 0, len(result.Tools))
	for _, t := range result.Tools {
		tool := SerializedTool{
			Name:        BuildMcpToolName(serverName, t.Name),
			Description: t.Description,
			IsMcp:       true,
			OriginalToolName: t.Name,
		}
		if t.InputSchema.Type != "" {
			schema := map[string]interface{}{
				"type":       t.InputSchema.Type,
				"properties": t.InputSchema.Properties,
			}
			if len(t.InputSchema.Required) > 0 {
				schema["required"] = t.InputSchema.Required
			}
			raw, _ := json.Marshal(schema)
			tool.InputJSONSchema = raw
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// FetchResourcesForClient fetches all resources from a connected MCP client.
// Mirrors TS fetchResourcesForClient.
func FetchResourcesForClient(ctx context.Context, c *client.Client, serverName string) ([]ServerResource, error) {
	result, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list resources from %q: %w", serverName, err)
	}

	resources := make([]ServerResource, 0, len(result.Resources))
	for _, r := range result.Resources {
		res := ServerResource{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MimeType:    r.MIMEType,
			Server:      serverName,
		}
		if r.Annotations != nil {
			raw, _ := json.Marshal(r.Annotations)
			res.Annotations = raw
		}
		resources = append(resources, res)
	}
	return resources, nil
}

// FetchCommandsForClient fetches all prompts from a connected MCP client,
// converting MCP prompts to Claude Code commands.
// Mirrors TS fetchCommandsForClient.
func FetchCommandsForClient(ctx context.Context, c *client.Client, serverName string) ([]MCPServerCommand, error) {
	result, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list prompts from %q: %w", serverName, err)
	}

	commands := make([]MCPServerCommand, 0, len(result.Prompts))
	for _, p := range result.Prompts {
		commands = append(commands, MCPServerCommand{
			Name:        BuildMcpToolName(serverName, p.Name),
			Description: p.Description,
			ServerName:  serverName,
		})
	}
	return commands, nil
}

// FetchAllFromServer fetches tools, commands, and resources from a single server.
func FetchAllFromServer(ctx context.Context, state *ClientState) (*DiscoveryResult, error) {
	var (
		tools     []SerializedTool
		commands  []MCPServerCommand
		resources []ServerResource
		mu        sync.Mutex
		errs      []error
		wg        sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		t, err := FetchToolsForClient(ctx, state.Client, state.Name)
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			tools = t
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		c, err := FetchCommandsForClient(ctx, state.Client, state.Name)
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			commands = c
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		r, err := FetchResourcesForClient(ctx, state.Client, state.Name)
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			resources = r
		}
		mu.Unlock()
	}()

	wg.Wait()

	if len(errs) > 0 {
		return &DiscoveryResult{
			Tools:     tools,
			Commands:  commands,
			Resources: resources,
		}, fmt.Errorf("partial failures: %v", errs)
	}

	return &DiscoveryResult{
		Tools:     tools,
		Commands:  commands,
		Resources: resources,
	}, nil
}

// CallMCPTool invokes a tool on a connected MCP server.
// Mirrors TS callMCPToolWithUrlElicitationRetry (simplified, no URL elicitation).
func (cm *ClientManager) CallMCPTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	state := cm.GetClient(serverName)
	if state == nil {
		return nil, fmt.Errorf("MCP server %q not connected", serverName)
	}

	callReq := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	callReq.Params.Name = toolName
	if args != nil {
		callReq.Params.Arguments = args
	}

	result, err := state.Client.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("call tool %q on %q: %w", toolName, serverName, err)
	}

	return result, nil
}

// ReadResource reads a resource from a connected MCP server.
func (cm *ClientManager) ReadResource(ctx context.Context, serverName, uri string) (*mcp.ReadResourceResult, error) {
	state := cm.GetClient(serverName)
	if state == nil {
		return nil, fmt.Errorf("MCP server %q not connected", serverName)
	}

	req := mcp.ReadResourceRequest{
		Request: mcp.Request{
			Method: "resources/read",
		},
	}
	req.Params.URI = uri

	result, err := state.Client.ReadResource(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("read resource %q from %q: %w", uri, serverName, err)
	}
	return result, nil
}
