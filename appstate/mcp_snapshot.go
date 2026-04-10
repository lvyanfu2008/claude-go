package appstate

import (
	"encoding/json"

	"goc/types"
)

// MCPServer connection kinds (src/services/mcp/types.ts MCPServerConnection.type).
const (
	MCPConnConnected = "connected"
	MCPConnFailed    = "failed"
	MCPConnNeedsAuth = "needs-auth"
	MCPConnPending   = "pending"
	MCPConnDisabled  = "disabled"
)

// MCPServerConnectionSnapshot is the JSON-safe subset of src/services/mcp/types.ts MCPServerConnection
// (omits live client handle and cleanup function).
type MCPServerConnectionSnapshot struct {
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	Capabilities         json.RawMessage `json:"capabilities,omitempty"`
	ServerInfo           json.RawMessage `json:"serverInfo,omitempty"`
	Instructions         string          `json:"instructions,omitempty"`
	Config               json.RawMessage `json:"config,omitempty"`
	Error                string          `json:"error,omitempty"`
	ReconnectAttempt     *int            `json:"reconnectAttempt,omitempty"`
	MaxReconnectAttempts *int            `json:"maxReconnectAttempts,omitempty"`
}

// MCPSerializedTool mirrors src/services/mcp/types.ts SerializedTool.
type MCPSerializedTool struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	InputJSONSchema  json.RawMessage `json:"inputJSONSchema,omitempty"`
	IsMcp            *bool           `json:"isMcp,omitempty"`
	OriginalToolName string          `json:"originalToolName,omitempty"`
}

// MCPServerResourceSnapshot mirrors src/services/mcp/types.ts ServerResource (Resource & { server }).
// Extra Resource fields from the MCP SDK are ignored on decode unless added here.
type MCPServerResourceSnapshot struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	MimeType    string          `json:"mimeType,omitempty"`
	Size        *int64          `json:"size,omitempty"`
	Annotations json.RawMessage `json:"annotations,omitempty"`
	Server      string          `json:"server"`
}

// McpState mirrors AppState.mcp with JSON-serializable client/tool shapes.
type McpState struct {
	Clients            []MCPServerConnectionSnapshot          `json:"clients"`
	Tools              []MCPSerializedTool                    `json:"tools"`
	Commands           []types.Command                        `json:"commands"`
	Resources          map[string][]MCPServerResourceSnapshot `json:"resources"`
	PluginReconnectKey int                                    `json:"pluginReconnectKey"`
}

// EmptyMcpState matches getDefaultAppState mcp default.
func EmptyMcpState() McpState {
	return McpState{
		Clients:            []MCPServerConnectionSnapshot{},
		Tools:              []MCPSerializedTool{},
		Commands:           []types.Command{},
		Resources:          make(map[string][]MCPServerResourceSnapshot),
		PluginReconnectKey: 0,
	}
}

// MarshalJSON normalizes nil slices and resources map.
func (m McpState) MarshalJSON() ([]byte, error) {
	type out struct {
		Clients            []MCPServerConnectionSnapshot          `json:"clients"`
		Tools              []MCPSerializedTool                    `json:"tools"`
		Commands           []types.Command                        `json:"commands"`
		Resources          map[string][]MCPServerResourceSnapshot `json:"resources"`
		PluginReconnectKey int                                    `json:"pluginReconnectKey"`
	}
	cl := m.Clients
	if cl == nil {
		cl = []MCPServerConnectionSnapshot{}
	}
	tl := m.Tools
	if tl == nil {
		tl = []MCPSerializedTool{}
	}
	cmd := m.Commands
	if cmd == nil {
		cmd = []types.Command{}
	}
	res := m.Resources
	if res == nil {
		res = make(map[string][]MCPServerResourceSnapshot)
	} else {
		res = cloneNormalizedMcpResourceMap(res)
	}
	return json.Marshal(out{
		Clients: cl, Tools: tl, Commands: cmd, Resources: res,
		PluginReconnectKey: m.PluginReconnectKey,
	})
}

// UnmarshalJSON normalizes nil collections.
func (m *McpState) UnmarshalJSON(data []byte) error {
	var s struct {
		Clients            []MCPServerConnectionSnapshot          `json:"clients"`
		Tools              []MCPSerializedTool                    `json:"tools"`
		Commands           []types.Command                        `json:"commands"`
		Resources          map[string][]MCPServerResourceSnapshot `json:"resources"`
		PluginReconnectKey int                                    `json:"pluginReconnectKey"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*m = McpState{
		Clients:            s.Clients,
		Tools:              s.Tools,
		Commands:           s.Commands,
		Resources:          s.Resources,
		PluginReconnectKey: s.PluginReconnectKey,
	}
	if m.Clients == nil {
		m.Clients = []MCPServerConnectionSnapshot{}
	}
	if m.Tools == nil {
		m.Tools = []MCPSerializedTool{}
	}
	if m.Commands == nil {
		m.Commands = []types.Command{}
	}
	if m.Resources == nil {
		m.Resources = make(map[string][]MCPServerResourceSnapshot)
	} else {
		m.Resources = normalizeMcpResourceMap(m.Resources)
	}
	return nil
}

func normalizeMcpResourceMap(m map[string][]MCPServerResourceSnapshot) map[string][]MCPServerResourceSnapshot {
	if m == nil {
		return nil
	}
	for k, v := range m {
		if v == nil {
			m[k] = []MCPServerResourceSnapshot{}
		}
	}
	return m
}

func cloneNormalizedMcpResourceMap(m map[string][]MCPServerResourceSnapshot) map[string][]MCPServerResourceSnapshot {
	out := make(map[string][]MCPServerResourceSnapshot, len(m))
	for k, v := range m {
		if v == nil {
			out[k] = []MCPServerResourceSnapshot{}
		} else {
			out[k] = v
		}
	}
	return out
}
