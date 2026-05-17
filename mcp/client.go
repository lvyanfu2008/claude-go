package mcp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ClientManager manages MCP server connections — connect, disconnect, reconnect, cache.
// Mirrors TS src/services/mcp/client.ts connectToServer / ensureConnectedClient.
type ClientManager struct {
	mu      sync.RWMutex
	clients map[string]*ClientState // server name → live client state
}

// ClientState holds a live MCP client and its metadata.
type ClientState struct {
	Client       *client.Client
	Name         string
	Config       ScopedMcpServerConfig
	Capabilities mcp.ServerCapabilities
	ServerInfo   *ServerInfo
	Instructions string
	ConnectedAt  time.Time
}

// NewClientManager creates a new ClientManager.
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*ClientState),
	}
}

// ConnectToServer connects to an MCP server based on its config.
// Returns the client state or an error.
func (cm *ClientManager) ConnectToServer(ctx context.Context, name string, scfg ScopedMcpServerConfig) (*ClientState, error) {
	normalizedName := NormalizeMcpServerName(name)

	// Check if already connected.
	cm.mu.RLock()
	if existing, ok := cm.clients[normalizedName]; ok {
		cm.mu.RUnlock()
		// Verify still alive with a ping.
		if err := existing.Client.Ping(ctx); err == nil {
			return existing, nil
		}
		// Dead connection — reconnect below.
		cm.mu.RUnlock()
		cm.DisconnectServer(normalizedName)
	} else {
		cm.mu.RUnlock()
	}

	var (
		c   *client.Client
		err error
	)

	switch cfg := scfg.Config.(type) {
	case McpStdioServerConfig:
		c, err = cm.connectStdio(cfg)
	case McpSSEServerConfig:
		c, err = cm.connectSSE(ctx, cfg)
	case McpSSEIDEServerConfig:
		c, err = cm.connectSSEIDE(ctx, cfg)
	case McpWebSocketIDEServerConfig:
		c, err = cm.connectWSIDE(cfg)
	case McpWebSocketServerConfig:
		c, err = cm.connectWebSocket(ctx, cfg)
	case McpHTTPServerConfig:
		c, err = cm.connectStreamableHTTP(ctx, cfg)
	case McpSdkServerConfig:
		c, err = cm.connectSDK(cfg)
	case McpClaudeAIProxyServerConfig:
		c, err = cm.connectClaudeAIProxy(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported MCP transport type: %T", cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("connect to MCP server %q: %w", name, err)
	}

	// Initialize the MCP session.
	initReq := mcp.InitializeRequest{
		Request: mcp.Request{
			Method: "initialize",
		},
	}
	initReq.Params.ProtocolVersion = "2025-06-18"
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "claude-code-go",
		Version: "0.1.0",
	}

	initResult, err := c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize MCP server %q: %w", name, err)
	}

	state := &ClientState{
		Client:       c,
		Name:         normalizedName,
		Config:       scfg,
		Capabilities: initResult.Capabilities,
		ConnectedAt:  time.Now(),
	}

	if initResult.ServerInfo.Name != "" {
		state.ServerInfo = &ServerInfo{
			Name:    initResult.ServerInfo.Name,
			Version: initResult.ServerInfo.Version,
		}
	}
	state.Instructions = initResult.Instructions

	cm.mu.Lock()
	cm.clients[normalizedName] = state
	cm.mu.Unlock()

	return state, nil
}

func (cm *ClientManager) connectStdio(cfg McpStdioServerConfig) (*client.Client, error) {
	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	c, err := client.NewStdioMCPClientWithOptions(cfg.Command, env, cfg.Args)
	if err != nil {
		return nil, fmt.Errorf("stdio transport: %w", err)
	}
	return c, nil
}

func (cm *ClientManager) connectSSE(ctx context.Context, cfg McpSSEServerConfig) (*client.Client, error) {
	opts := []transport.ClientOption{}
	if len(cfg.Headers) > 0 {
		opts = append(opts, client.WithHeaders(cfg.Headers))
	}
	c, err := client.NewSSEMCPClient(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("sse transport: %w", err)
	}
	return c, nil
}

func (cm *ClientManager) connectStreamableHTTP(ctx context.Context, cfg McpHTTPServerConfig) (*client.Client, error) {
	opts := []transport.StreamableHTTPCOption{}
	if len(cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(cfg.Headers))
	}
	c, err := client.NewStreamableHttpClient(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("streamable http transport: %w", err)
	}
	return c, nil
}

// DisconnectServer disconnects and removes a server from the cache.
func (cm *ClientManager) DisconnectServer(name string) {
	normalizedName := NormalizeMcpServerName(name)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if state, ok := cm.clients[normalizedName]; ok {
		state.Client.Close()
		delete(cm.clients, normalizedName)
	}
}

// GetClient returns the client state for a server, or nil if not connected.
func (cm *ClientManager) GetClient(name string) *ClientState {
	normalizedName := NormalizeMcpServerName(name)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.clients[normalizedName]
}

// ListClients returns all connected client names.
func (cm *ClientManager) ListClients() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	names := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		names = append(names, name)
	}
	return names
}

// CloseAll disconnects all servers.
func (cm *ClientManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for name, state := range cm.clients {
		state.Client.Close()
		log.Printf("[mcp] disconnected server %q", name)
	}
	cm.clients = make(map[string]*ClientState)
}

// ReconnectServer reconnects to a server and refreshes its tools/resources.
func (cm *ClientManager) ReconnectServer(ctx context.Context, name string) (*ClientState, error) {
	cm.DisconnectServer(name)
	return cm.ConnectToServer(ctx, name, cm.getServerConfig(name))
}

func (cm *ClientManager) getServerConfig(name string) ScopedMcpServerConfig {
	normalizedName := NormalizeMcpServerName(name)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if state, ok := cm.clients[normalizedName]; ok {
		return state.Config
	}
	return ScopedMcpServerConfig{}
}

// BuildMcpToolName builds the full tool name for an MCP tool: mcp__<server>__<tool>.
// Mirrors TS services/mcp/mcpStringUtils.ts buildMcpToolName.
func BuildMcpToolName(serverName, toolName string) string {
	return fmt.Sprintf("mcp__%s__%s", NormalizeMcpServerName(serverName), toolName)
}

// GetMcpPrefix extracts the MCP prefix "mcp__<server>" from a full tool name.
func GetMcpPrefix(serverName string) string {
	return "mcp__" + NormalizeMcpServerName(serverName)
}

// IsMcpTool checks if a tool name starts with "mcp__".
func IsMcpTool(toolName string) bool {
	return strings.HasPrefix(toolName, "mcp__")
}

// ParseMcpToolName extracts server name and tool name from full MCP tool name.
func ParseMcpToolName(fullName string) (serverName, toolName string, ok bool) {
	if !strings.HasPrefix(fullName, "mcp__") {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(fullName, "mcp__"), "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// connectSSEIDE connects to an SSE-IDE MCP server (SSE transport without auth).
func (cm *ClientManager) connectSSEIDE(ctx context.Context, cfg McpSSEIDEServerConfig) (*client.Client, error) {
	opts := []transport.ClientOption{}
	opts = append(opts, client.WithHeaders(map[string]string{
		"User-Agent": "claude-code-go",
	}))
	c, err := client.NewSSEMCPClient(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("sse-ide transport: %w", err)
	}
	return c, nil
}

// connectWSIDE connects to a WS-IDE MCP server (WebSocket transport with IDE auth).
func (cm *ClientManager) connectWSIDE(cfg McpWebSocketIDEServerConfig) (*client.Client, error) {
	headers := map[string]string{
		"User-Agent": "claude-code-go",
	}
	if cfg.AuthToken != "" {
		headers["X-Claude-Code-Ide-Authorization"] = cfg.AuthToken
	}
	wsTransport := NewWebSocketTransport(cfg.URL, headers)
	c := client.NewClient(wsTransport)
	return c, nil
}

// connectWebSocket connects to a WebSocket MCP server.
func (cm *ClientManager) connectWebSocket(ctx context.Context, cfg McpWebSocketServerConfig) (*client.Client, error) {
	headers := make(map[string]string)
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	headers["User-Agent"] = "claude-code-go"
	wsTransport := NewWebSocketTransport(cfg.URL, headers)
	c := client.NewClient(wsTransport)
	return c, nil
}

// connectClaudeAIProxy connects to a claude.ai proxy MCP server using streamable HTTP.
func (cm *ClientManager) connectClaudeAIProxy(ctx context.Context, cfg McpClaudeAIProxyServerConfig) (*client.Client, error) {
	proxyURL := buildClaudeAIProxyURL(cfg.ID)
	opts := []transport.StreamableHTTPCOption{}
	headers := map[string]string{
		"User-Agent":              "claude-code-go",
		"X-Mcp-Client-Session-Id": cm.getSessionID(),
	}
	// Add OAuth bearer token if available.
	if token := getClaudeAIOAuthToken(); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	opts = append(opts, transport.WithHTTPHeaders(headers))
	c, err := client.NewStreamableHttpClient(proxyURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("claudeai-proxy transport: %w", err)
	}
	return c, nil
}

// getSessionID returns a session ID string for the client manager.
func (cm *ClientManager) getSessionID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, state := range cm.clients {
		// Return first connected client's name as a proxy for session ID.
		return state.Name
	}
	return ""
}
