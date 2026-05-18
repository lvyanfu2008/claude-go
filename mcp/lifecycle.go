package mcp

import (
	"context"
	"encoding/json"
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

	// OAuth support
	oauthManager  *OAuthManager
	tokenStore    *TokenStore
	needsAuthCache map[string]time.Time // server name → cache expiry

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
		oauthManager:         NewOAuthManager(),
		tokenStore:           NewTokenStore(),
		needsAuthCache:       make(map[string]time.Time),
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
	normalized := NormalizeMcpServerName(name)

	// Check if this server needs OAuth authentication.
	if HasOAuthConfig(scfg.Config) && !cmgr.hasValidToken(normalized) {
		cmgr.mu.Lock()
		cmgr.needsAuthCache[normalized] = time.Now().Add(15 * time.Minute)
		// Inject the McpAuth pseudo-tool into discovery.
		cmgr.discovery[normalized] = cmgr.buildAuthDiscovery(name, scfg)
		cmgr.mu.Unlock()
		log.Printf("[mcp] server %q needs OAuth auth, injected auth pseudo-tool", name)
		return nil
	}

	state, err := cmgr.clientMgr.ConnectToServer(ctx, name, scfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	discovery, err := FetchAllFromServer(ctx, state)
	if err != nil {
		log.Printf("[mcp] partial discovery for %q: %v", name, err)
	}

	cmgr.mu.Lock()
	cmgr.discovery[normalized] = discovery
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
	cmgr.needsAuthCache = make(map[string]time.Time)
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

// HasOAuthConfig reports whether an MCP server config includes OAuth settings.
func HasOAuthConfig(config McpServerConfig) bool {
	switch c := config.(type) {
	case McpSSEServerConfig:
		return c.OAuth != nil
	case McpHTTPServerConfig:
		return c.OAuth != nil
	default:
		return false
	}
}

// hasValidToken checks whether a non-expired OAuth token exists for the server.
func (cmgr *ConnectionManager) hasValidToken(serverName string) bool {
	tok := cmgr.tokenStore.Get(serverName)
	if tok == nil {
		return false
	}
	return tok.Valid()
}

// buildAuthDiscovery creates a DiscoveryResult containing only the McpAuth pseudo-tool.
func (cmgr *ConnectionManager) buildAuthDiscovery(name string, scfg ScopedMcpServerConfig) *DiscoveryResult {
	config := scfg.Config
	serverName := NormalizeMcpServerName(name)

	// Build description similar to TS createMcpAuthTool.
	transport := config.ConfigType()
	desc := fmt.Sprintf(
		"The `%s` MCP server (%s) is installed but requires authentication. "+
			"Call this tool to start the OAuth flow — you'll receive an authorization URL to share with the user. "+
			"Once the user completes authorization in their browser, the server's real tools will become available automatically.",
		serverName, transport,
	)

	schema := json.RawMessage([]byte(
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{},"additionalProperties":false}`,
	))

	authTool := SerializedTool{
		Name:             BuildMcpToolName(serverName, "authenticate"),
		Description:      desc,
		InputJSONSchema:  schema,
		IsMcp:            true,
		OriginalToolName: "authenticate",
	}

	return &DiscoveryResult{
		Tools: []SerializedTool{authTool},
	}
}

// HandleMcpOAuth starts the OAuth flow for a server and returns the authorization URL.
// It does NOT block — the flow runs in the background and reconnects on completion.
func (cmgr *ConnectionManager) HandleMcpOAuth(ctx context.Context, serverName string) (map[string]any, error) {
	normalized := NormalizeMcpServerName(serverName)

	cmgr.mu.RLock()
	scfg, ok := cmgr.serverConfigs[normalized]
	cmgr.mu.RUnlock()
	if !ok {
		return map[string]any{
			"status":  "error",
			"message": fmt.Sprintf("Server %q not found in configuration.", serverName),
		}, nil
	}

	config := scfg.Config
	if !HasOAuthConfig(config) {
		return map[string]any{
			"status":  "unsupported",
			"message": fmt.Sprintf("Server %q does not support OAuth authentication.", serverName),
		}, nil
	}

	// claudeai-proxy uses a separate UI-based auth flow.
	if _, ok := config.(McpClaudeAIProxyServerConfig); ok {
		return map[string]any{
			"status":  "unsupported",
			"message": fmt.Sprintf("This is a claude.ai MCP connector. Run /mcp and select %q to authenticate.", serverName),
		}, nil
	}

	var oauthCfg *McpOAuthConfig
	switch c := config.(type) {
	case McpSSEServerConfig:
		oauthCfg = c.OAuth
	case McpHTTPServerConfig:
		oauthCfg = c.OAuth
	}

	if oauthCfg == nil || oauthCfg.AuthServerMetadataURL == "" {
		return map[string]any{
			"status":  "error",
			"message": fmt.Sprintf("Server %q has incomplete OAuth configuration.", serverName),
		}, nil
	}

	clientID := oauthCfg.ClientID
	callbackPort := oauthCfg.CallbackPort
	if callbackPort <= 0 {
		callbackPort = 9876
	}

	// Start the OAuth flow in a background goroutine. We need to capture the
	// authorization URL to return it immediately. For now, we start the full
	// synchronous flow in the background and notify the caller that OAuth has
	// been initiated. The user will need to watch their browser.
	go func() {
		token, err := cmgr.oauthManager.PerformOAuthFlow(
			context.Background(), serverName, oauthCfg.AuthServerMetadataURL, clientID, callbackPort,
		)
		if err != nil {
			log.Printf("[mcp] OAuth failed for %q: %v", serverName, err)
			return
		}
		cmgr.tokenStore.Set(normalized, token)
		cmgr.ClearNeedsAuthCache(normalized)

		// Reconnect the server with the new token.
		if err := cmgr.reconnectAfterOAuth(context.Background(), normalized, scfg); err != nil {
			log.Printf("[mcp] reconnect after OAuth failed for %q: %v", serverName, err)
		}
	}()

	return map[string]any{
		"status":  "auth_url",
		"message": fmt.Sprintf("OAuth flow started for %q. A browser window should open. Complete authorization there and the server will reconnect automatically.", serverName),
	}, nil
}

// reconnectAfterOAuth reconnects a server after successful OAuth authentication.
func (cmgr *ConnectionManager) reconnectAfterOAuth(ctx context.Context, normalizedName string, scfg ScopedMcpServerConfig) error {
	state, err := cmgr.clientMgr.ReconnectServer(ctx, normalizedName)
	if err != nil {
		return fmt.Errorf("reconnect: %w", err)
	}

	discovery, err := FetchAllFromServer(ctx, state)
	if err != nil {
		log.Printf("[mcp] partial discovery after OAuth for %q: %v", normalizedName, err)
	}

	cmgr.mu.Lock()
	cmgr.discovery[normalizedName] = discovery
	cmgr.mu.Unlock()

	log.Printf("[mcp] server %q reconnected after OAuth with %d tools", normalizedName, len(discovery.Tools))
	return nil
}

// IsNeedsAuthCached reports whether a server is in the needs-auth cache.
func (cmgr *ConnectionManager) IsNeedsAuthCached(serverName string) bool {
	cmgr.mu.RLock()
	defer cmgr.mu.RUnlock()
	expiry, ok := cmgr.needsAuthCache[NormalizeMcpServerName(serverName)]
	if !ok {
		return false
	}
	return time.Now().Before(expiry)
}

// ClearNeedsAuthCache removes a server from the needs-auth cache.
func (cmgr *ConnectionManager) ClearNeedsAuthCache(serverName string) {
	cmgr.mu.Lock()
	defer cmgr.mu.Unlock()
	delete(cmgr.needsAuthCache, NormalizeMcpServerName(serverName))
}
