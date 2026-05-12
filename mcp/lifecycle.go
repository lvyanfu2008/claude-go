package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// ConnectionManager handles the lifecycle of multiple MCP server connections.
// Mirrors TS useManageMCPConnections.
type ConnectionManager struct {
	clientMgr    *ClientManager
	serverConfigs map[string]ScopedMcpServerConfig
	discovery    map[string]*DiscoveryResult // server name → discovered tools/commands/resources
	mu           sync.RWMutex

	// Reconnect policy
	maxReconnectAttempts int
	reconnectBackoff     time.Duration
}

// NewConnectionManager creates a new ConnectionManager.
func NewConnectionManager(clientMgr *ClientManager) *ConnectionManager {
	return &ConnectionManager{
		clientMgr:            clientMgr,
		serverConfigs:        make(map[string]ScopedMcpServerConfig),
		discovery:            make(map[string]*DiscoveryResult),
		maxReconnectAttempts: 3,
		reconnectBackoff:     2 * time.Second,
	}
}

// AddServer adds a server config to be managed.
func (cmgr *ConnectionManager) AddServer(name string, scfg ScopedMcpServerConfig) {
	cmgr.mu.Lock()
	defer cmgr.mu.Unlock()
	cmgr.serverConfigs[NormalizeMcpServerName(name)] = scfg
}

// RemoveServer removes a server config and disconnects it.
func (cmgr *ConnectionManager) RemoveServer(name string) {
	normalized := NormalizeMcpServerName(name)
	cmgr.mu.Lock()
	delete(cmgr.serverConfigs, normalized)
	delete(cmgr.discovery, normalized)
	cmgr.mu.Unlock()
	cmgr.clientMgr.DisconnectServer(normalized)
}

// StartAll connects to all configured servers and discovers their capabilities.
func (cmgr *ConnectionManager) StartAll(ctx context.Context) error {
	cmgr.mu.RLock()
	configs := make(map[string]ScopedMcpServerConfig, len(cmgr.serverConfigs))
	for name, cfg := range cmgr.serverConfigs {
		configs[name] = cfg
	}
	cmgr.mu.RUnlock()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for name, scfg := range configs {
		wg.Add(1)
		go func(serverName string, cfg ScopedMcpServerConfig) {
			defer wg.Done()
			if err := cmgr.startServer(ctx, serverName, cfg); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("server %q: %w", serverName, err))
				mu.Unlock()
			}
		}(name, scfg)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("MCP connection errors: %v", errs)
	}
	return nil
}

func (cmgr *ConnectionManager) startServer(ctx context.Context, name string, scfg ScopedMcpServerConfig) error {
	state, err := cmgr.clientMgr.ConnectToServer(ctx, name, scfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	discovery, err := FetchAllFromServer(ctx, state)
	if err != nil {
		log.Printf("[mcp] partial discovery for %q: %v", name, err)
	}

	cmgr.mu.Lock()
	cmgr.discovery[NormalizeMcpServerName(name)] = discovery
	cmgr.mu.Unlock()

	return nil
}

// GetAllTools returns all discovered MCP tools across all connected servers.
func (cmgr *ConnectionManager) GetAllTools() []SerializedTool {
	cmgr.mu.RLock()
	defer cmgr.mu.RUnlock()
	var all []SerializedTool
	for _, d := range cmgr.discovery {
		if d != nil {
			all = append(all, d.Tools...)
		}
	}
	return all
}

// GetAllResources returns all discovered MCP resources across all connected servers.
func (cmgr *ConnectionManager) GetAllResources() map[string][]ServerResource {
	cmgr.mu.RLock()
	defer cmgr.mu.RUnlock()
	result := make(map[string][]ServerResource, len(cmgr.discovery))
	for name, d := range cmgr.discovery {
		if d != nil && len(d.Resources) > 0 {
			result[name] = d.Resources
		}
	}
	return result
}

// GetServerStatus returns the connection status of all servers.
func (cmgr *ConnectionManager) GetServerStatus() map[string]MCPServerStatus {
	cmgr.mu.RLock()
	defer cmgr.mu.RUnlock()
	result := make(map[string]MCPServerStatus, len(cmgr.serverConfigs))
	for name := range cmgr.serverConfigs {
		state := cmgr.clientMgr.GetClient(name)
		if state != nil {
			result[name] = StatusConnected
		} else {
			result[name] = StatusPending
		}
	}
	return result
}

// Shutdown disconnects all servers.
func (cmgr *ConnectionManager) Shutdown() {
	cmgr.mu.Lock()
	cmgr.discovery = make(map[string]*DiscoveryResult)
	cmgr.serverConfigs = make(map[string]ScopedMcpServerConfig)
	cmgr.mu.Unlock()
	cmgr.clientMgr.CloseAll()
}

// ExecuteMCPTool executes an MCP tool by its full name (mcp__server__tool).
func (cmgr *ConnectionManager) ExecuteMCPTool(ctx context.Context, fullToolName string, args map[string]interface{}) (string, error) {
	serverName, toolName, ok := ParseMcpToolName(fullToolName)
	if !ok {
		return "", fmt.Errorf("not an MCP tool: %q", fullToolName)
	}

	result, err := cmgr.clientMgr.CallMCPTool(ctx, serverName, toolName, args)
	if err != nil {
		return "", err
	}

	return ProcessMCPResult(result), nil
}

// ExecuteMCPToolStreaming executes an MCP tool with streaming result handling.
func (cmgr *ConnectionManager) ExecuteMCPToolStreaming(ctx context.Context, fullToolName string, args map[string]interface{}) (string, error) {
	serverName, toolName, ok := ParseMcpToolName(fullToolName)
	if !ok {
		return "", fmt.Errorf("not an MCP tool: %q", fullToolName)
	}

	state := cmgr.clientMgr.GetClient(serverName)
	if state == nil {
		return "", fmt.Errorf("MCP server %q not connected", serverName)
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
		return "", fmt.Errorf("call tool %q on %q: %w", toolName, serverName, err)
	}

	return ProcessMCPResult(result), nil
}

