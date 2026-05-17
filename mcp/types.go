// Package mcp ports src/services/mcp/ — MCP server connection management,
// configuration, tool discovery, and OAuth authentication.
package mcp

import "encoding/json"

// ConfigScope mirrors src/services/mcp/types.ts ConfigScope.
type ConfigScope string

const (
	ScopeLocal      ConfigScope = "local"
	ScopeUser       ConfigScope = "user"
	ScopeProject    ConfigScope = "project"
	ScopeDynamic    ConfigScope = "dynamic"
	ScopeEnterprise ConfigScope = "enterprise"
	ScopeClaudeAI   ConfigScope = "claudeai"
	ScopeManaged    ConfigScope = "managed"
)

// Transport mirrors src/services/mcp/types.ts Transport.
type Transport string

const (
	TransportStdio         Transport = "stdio"
	TransportSSE           Transport = "sse"
	TransportSSEIDE        Transport = "sse-ide"
	TransportWSIDE         Transport = "ws-ide"
	TransportHTTP          Transport = "http"
	TransportWS            Transport = "ws"
	TransportSDK           Transport = "sdk"
	TransportClaudeAIProxy Transport = "claudeai-proxy"
)

// McpOAuthConfig mirrors src/services/mcp/types.ts McpOAuthConfig.
type McpOAuthConfig struct {
	ClientID               string `json:"clientId,omitempty"`
	CallbackPort           int    `json:"callbackPort,omitempty"`
	AuthServerMetadataURL  string `json:"authServerMetadataUrl,omitempty"`
	XAA                    *bool  `json:"xaa,omitempty"`
}

// McpStdioServerConfig mirrors src/services/mcp/types.ts McpStdioServerConfig.
type McpStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // "stdio", optional for backwards compat
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig mirrors src/services/mcp/types.ts McpSSEServerConfig.
type McpSSEServerConfig struct {
	Type          string            `json:"type"` // "sse"
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

// McpSSEIDEServerConfig mirrors src/services/mcp/types.ts McpSSEIDEServerConfig.
type McpSSEIDEServerConfig struct {
	Type                string `json:"type"` // "sse-ide"
	URL                 string `json:"url"`
	IDEName             string `json:"ideName"`
	IDERunningInWindows bool   `json:"ideRunningInWindows,omitempty"`
}

// McpWebSocketIDEServerConfig mirrors src/services/mcp/types.ts McpWebSocketIDEServerConfig.
type McpWebSocketIDEServerConfig struct {
	Type                string `json:"type"` // "ws-ide"
	URL                 string `json:"url"`
	IDEName             string `json:"ideName"`
	AuthToken           string `json:"authToken,omitempty"`
	IDERunningInWindows bool   `json:"ideRunningInWindows,omitempty"`
}

// McpHTTPServerConfig mirrors src/services/mcp/types.ts McpHTTPServerConfig.
type McpHTTPServerConfig struct {
	Type          string            `json:"type"` // "http"
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
	OAuth         *McpOAuthConfig   `json:"oauth,omitempty"`
}

// McpWebSocketServerConfig mirrors src/services/mcp/types.ts McpWebSocketServerConfig.
type McpWebSocketServerConfig struct {
	Type          string            `json:"type"` // "ws"
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	HeadersHelper string            `json:"headersHelper,omitempty"`
}

// McpSdkServerConfig mirrors src/services/mcp/types.ts McpSdkServerConfig.
type McpSdkServerConfig struct {
	Type string `json:"type"` // "sdk"
	Name string `json:"name"`
}

// McpClaudeAIProxyServerConfig mirrors src/services/mcp/types.ts McpClaudeAIProxyServerConfig.
type McpClaudeAIProxyServerConfig struct {
	Type string `json:"type"` // "claudeai-proxy"
	URL  string `json:"url"`
	ID   string `json:"id"`
}

// McpServerConfig is the union of all MCP server config types.
// Use type switch on the Type field to determine the concrete config.
type McpServerConfig interface {
	mcpServerConfigMarker()
	ConfigType() string
}

func (c McpStdioServerConfig) mcpServerConfigMarker()          {}
func (c McpStdioServerConfig) ConfigType() string               { return c.Type }
func (c McpSSEServerConfig) mcpServerConfigMarker()             {}
func (c McpSSEServerConfig) ConfigType() string                 { return "sse" }
func (c McpSSEIDEServerConfig) mcpServerConfigMarker()          {}
func (c McpSSEIDEServerConfig) ConfigType() string              { return "sse-ide" }
func (c McpWebSocketIDEServerConfig) mcpServerConfigMarker()    {}
func (c McpWebSocketIDEServerConfig) ConfigType() string        { return "ws-ide" }
func (c McpHTTPServerConfig) mcpServerConfigMarker()            {}
func (c McpHTTPServerConfig) ConfigType() string                { return "http" }
func (c McpWebSocketServerConfig) mcpServerConfigMarker()       {}
func (c McpWebSocketServerConfig) ConfigType() string           { return "ws" }
func (c McpSdkServerConfig) mcpServerConfigMarker()             {}
func (c McpSdkServerConfig) ConfigType() string                 { return "sdk" }
func (c McpClaudeAIProxyServerConfig) mcpServerConfigMarker()   {}
func (c McpClaudeAIProxyServerConfig) ConfigType() string       { return "claudeai-proxy" }

// ScopedMcpServerConfig mirrors src/services/mcp/types.ts ScopedMcpServerConfig.
type ScopedMcpServerConfig struct {
	Config McpServerConfig `json:"config"`
	Scope  ConfigScope     `json:"scope"`

	// PluginSource is the providing plugin's source identifier (e.g. "slack@anthropic").
	PluginSource string `json:"pluginSource,omitempty"`
}

// McpJsonConfig mirrors src/services/mcp/types.ts McpJsonConfig.
type McpJsonConfig struct {
	McpServers map[string]json.RawMessage `json:"mcpServers"`
}

// MCPServerStatus is the connection status of an MCP server.
type MCPServerStatus string

const (
	StatusConnected  MCPServerStatus = "connected"
	StatusFailed     MCPServerStatus = "failed"
	StatusNeedsAuth  MCPServerStatus = "needs-auth"
	StatusPending    MCPServerStatus = "pending"
	StatusDisabled   MCPServerStatus = "disabled"
)

// ConnectedMCPServer mirrors src/services/mcp/types.ts ConnectedMCPServer.
type ConnectedMCPServer struct {
	Name         string               `json:"name"`
	Status       MCPServerStatus      `json:"type"` // "connected"
	Capabilities json.RawMessage      `json:"capabilities,omitempty"`
	ServerInfo   *ServerInfo          `json:"serverInfo,omitempty"`
	Instructions string               `json:"instructions,omitempty"`
	Config       ScopedMcpServerConfig `json:"config"`
}

// ServerInfo mirrors the MCP SDK server info.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// FailedMCPServer mirrors src/services/mcp/types.ts FailedMCPServer.
type FailedMCPServer struct {
	Name   string               `json:"name"`
	Status MCPServerStatus      `json:"type"` // "failed"
	Config ScopedMcpServerConfig `json:"config"`
	Error  string               `json:"error,omitempty"`
}

// NeedsAuthMCPServer mirrors src/services/mcp/types.ts NeedsAuthMCPServer.
type NeedsAuthMCPServer struct {
	Name   string               `json:"name"`
	Status MCPServerStatus      `json:"type"` // "needs-auth"
	Config ScopedMcpServerConfig `json:"config"`
}

// PendingMCPServer mirrors src/services/mcp/types.ts PendingMCPServer.
type PendingMCPServer struct {
	Name                string               `json:"name"`
	Status              MCPServerStatus      `json:"type"` // "pending"
	Config              ScopedMcpServerConfig `json:"config"`
	ReconnectAttempt    int                  `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts int                 `json:"maxReconnectAttempts,omitempty"`
}

// DisabledMCPServer mirrors src/services/mcp/types.ts DisabledMCPServer.
type DisabledMCPServer struct {
	Name   string               `json:"name"`
	Status MCPServerStatus      `json:"type"` // "disabled"
	Config ScopedMcpServerConfig `json:"config"`
}

// MCPServerConnection is the union of all server connection states.
type MCPServerConnection interface {
	mcpServerConnectionMarker()
	ServerName() string
	ServerStatus() MCPServerStatus
}

func (c ConnectedMCPServer) mcpServerConnectionMarker()  {}
func (c ConnectedMCPServer) ServerName() string           { return c.Name }
func (c ConnectedMCPServer) ServerStatus() MCPServerStatus { return StatusConnected }
func (f FailedMCPServer) mcpServerConnectionMarker()     {}
func (f FailedMCPServer) ServerName() string              { return f.Name }
func (f FailedMCPServer) ServerStatus() MCPServerStatus    { return StatusFailed }
func (n NeedsAuthMCPServer) mcpServerConnectionMarker()  {}
func (n NeedsAuthMCPServer) ServerName() string           { return n.Name }
func (n NeedsAuthMCPServer) ServerStatus() MCPServerStatus { return StatusNeedsAuth }
func (p PendingMCPServer) mcpServerConnectionMarker()    {}
func (p PendingMCPServer) ServerName() string             { return p.Name }
func (p PendingMCPServer) ServerStatus() MCPServerStatus   { return StatusPending }
func (d DisabledMCPServer) mcpServerConnectionMarker()   {}
func (d DisabledMCPServer) ServerName() string            { return d.Name }
func (d DisabledMCPServer) ServerStatus() MCPServerStatus  { return StatusDisabled }

// SerializedTool mirrors src/services/mcp/types.ts SerializedTool.
type SerializedTool struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	InputJSONSchema  json.RawMessage `json:"inputJSONSchema,omitempty"`
	IsMcp            bool            `json:"isMcp,omitempty"`
	OriginalToolName string          `json:"originalToolName,omitempty"`
}

// SerializedClient mirrors src/services/mcp/types.ts SerializedClient.
type SerializedClient struct {
	Name         string          `json:"name"`
	Status       MCPServerStatus `json:"type"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
}

// ServerResource mirrors src/services/mcp/types.ts ServerResource.
type ServerResource struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	MimeType    string          `json:"mimeType,omitempty"`
	Size        *int64          `json:"size,omitempty"`
	Annotations json.RawMessage `json:"annotations,omitempty"`
	Server      string          `json:"server"`
}

// MCPCliState mirrors src/services/mcp/types.ts MCPCliState.
type MCPCliState struct {
	Clients          []SerializedClient            `json:"clients"`
	Configs          map[string]ScopedMcpServerConfig `json:"configs"`
	Tools            []SerializedTool              `json:"tools"`
	Resources        map[string][]ServerResource   `json:"resources"`
	NormalizedNames  map[string]string             `json:"normalizedNames,omitempty"`
}
