package mcp

import (
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/server"
)

// sdkRegistry holds in-process MCP server instances registered by name.
var (
	sdkRegistryMu sync.RWMutex
	sdkRegistry   = map[string]*server.MCPServer{}
)

// RegisterSDKServer registers an in-process MCP server by name.
func RegisterSDKServer(name string, srv *server.MCPServer) {
	sdkRegistryMu.Lock()
	defer sdkRegistryMu.Unlock()
	sdkRegistry[name] = srv
}

// GetSDKServer returns a registered in-process MCP server by name.
func GetSDKServer(name string) (*server.MCPServer, error) {
	sdkRegistryMu.RLock()
	defer sdkRegistryMu.RUnlock()
	srv, ok := sdkRegistry[name]
	if !ok {
		return nil, fmt.Errorf("sdk MCP server %q not registered", name)
	}
	return srv, nil
}

// UnregisterSDKServer removes a registered SDK server.
func UnregisterSDKServer(name string) {
	sdkRegistryMu.Lock()
	defer sdkRegistryMu.Unlock()
	delete(sdkRegistry, name)
}

// connectSDK connects to an in-process SDK MCP server.
func (cm *ClientManager) connectSDK(cfg McpSdkServerConfig) (*client.Client, error) {
	srv, err := GetSDKServer(cfg.Name)
	if err != nil {
		return nil, err
	}
	c, err := client.NewInProcessClient(srv)
	if err != nil {
		return nil, fmt.Errorf("sdk transport: %w", err)
	}
	return c, nil
}
