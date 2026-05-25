package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadMcpJsonConfig loads MCP server configs from a .mcp.json file.
// Mirrors TS writeMcpjsonFile / McpJsonConfigSchema.
func LoadMcpJsonConfig(cwd string) (*McpJsonConfig, error) {
	path := filepath.Join(cwd, ".mcp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read .mcp.json: %w", err)
	}
	var cfg McpJsonConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse .mcp.json: %w", err)
	}
	return &cfg, nil
}

// SaveMcpJsonConfig writes MCP config to .mcp.json file.
func SaveMcpJsonConfig(cwd string, cfg *McpJsonConfig) error {
	path := filepath.Join(cwd, ".mcp.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .mcp.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write .mcp.json: %w", err)
	}
	return nil
}

// ConfigSource represents where an MCP server config comes from.
type ConfigSource int

const (
	SourceUserSettings   ConfigSource = iota // ~/.harness/settings.go.json
	SourceProjectSettings                     // .harness/settings.go.json
	SourceLocalSettings                       // .harness/settings.local.json
	SourceMcpJson                             // .mcp.json
	SourceEnterprise                          // managed-mcp.json
	SourceDynamic                             // runtime-added
)

// ParsedMcpConfigs holds MCP server configs from a single source.
type ParsedMcpConfigs struct {
	Servers map[string]McpServerConfig
	Source  ConfigSource
	Scope   ConfigScope
}

// ParseMcpServersFromJSON parses "mcpServers" from a settings JSON blob.
// Each entry in the map is a raw JSON object that gets parsed into the correct config type.
func ParseMcpServersFromJSON(raw map[string]json.RawMessage) (map[string]McpServerConfig, error) {
	result := make(map[string]McpServerConfig, len(raw))
	for name, data := range raw {
		cfg, err := ParseMcpServerConfig(data)
		if err != nil {
			return nil, fmt.Errorf("server %q: %w", name, err)
		}
		result[name] = cfg
	}
	return result, nil
}

// ParseMcpServerConfig parses a single MCP server config from raw JSON.
func ParseMcpServerConfig(data json.RawMessage) (McpServerConfig, error) {
	// Peek at the "type" field to determine the config variant.
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("invalid MCP server config: %w", err)
	}

	switch typeCheck.Type {
	case "stdio", "":
		var c McpStdioServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		// Expand env vars in command, args, and env values.
		c.Command = ExpandEnvVars(c.Command)
		for i, arg := range c.Args {
			c.Args[i] = ExpandEnvVars(arg)
		}
		if c.Env != nil {
			for k, v := range c.Env {
				c.Env[k] = ExpandEnvVars(v)
			}
		}
		return c, nil
	case "sse":
		var c McpSSEServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		c.URL = ExpandEnvVars(c.URL)
		return c, nil
	case "sse-ide":
		var c McpSSEIDEServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case "ws-ide":
		var c McpWebSocketIDEServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case "http":
		var c McpHTTPServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		c.URL = ExpandEnvVars(c.URL)
		return c, nil
	case "ws":
		var c McpWebSocketServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		c.URL = ExpandEnvVars(c.URL)
		return c, nil
	case "sdk":
		var c McpSdkServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case "claudeai-proxy":
		var c McpClaudeAIProxyServerConfig
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown MCP server type: %q", typeCheck.Type)
	}
}

// GetServerConfigName returns a human-readable name for an MCP server config.
func GetServerConfigName(cfg McpServerConfig) string {
	switch c := cfg.(type) {
	case McpStdioServerConfig:
		return c.Command
	case McpSSEServerConfig, McpSSEIDEServerConfig, McpWebSocketIDEServerConfig, McpHTTPServerConfig, McpWebSocketServerConfig:
		return ""
	case McpSdkServerConfig:
		return c.Name
	case McpClaudeAIProxyServerConfig:
		return c.ID
	}
	return ""
}

// AddScopeToServers adds a scope to each server config in the map.
func AddScopeToServers(servers map[string]McpServerConfig, scope ConfigScope) map[string]ScopedMcpServerConfig {
	result := make(map[string]ScopedMcpServerConfig, len(servers))
	for name, cfg := range servers {
		result[name] = ScopedMcpServerConfig{Config: cfg, Scope: scope}
	}
	return result
}

// ServerConfigJSON serializes a McpServerConfig to display-friendly JSON.
func ServerConfigJSON(cfg McpServerConfig) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NormalizeMcpServerName normalizes a server name for display and comparison.
// Mirrors TS normalizeNameForMCP.
func NormalizeMcpServerName(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, strings.ToLower(strings.TrimSpace(name)))
}

// IsMcpServerDisabled checks if a server is explicitly disabled in a settings map.
// TS mirrors the h/disableMcpServers list in settings.
func IsMcpServerDisabled(name string, disabledServers []string) bool {
	for _, d := range disabledServers {
		if strings.EqualFold(d, name) {
			return true
		}
	}
	return false
}

// path helpers

// UserSettingsPath returns ~/.harness/settings.go.json or $CLAUDE_CONFIG_DIR/settings.go.json.
func UserSettingsPath() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return filepath.Join(d, "settings.go.json")
	}
	h, _ := os.UserHomeDir()
	if h == "" {
		return ""
	}
	return filepath.Join(h, ".harness", "settings.go.json")
}

// ProjectSettingsPath returns <cwd>/.harness/settings.go.json.
func ProjectSettingsPath(cwd string) string {
	return filepath.Join(cwd, ".harness", "settings.go.json")
}

// LocalSettingsPath returns <cwd>/.harness/settings.local.json.
func LocalSettingsPath(cwd string) string {
	return filepath.Join(cwd, ".harness", "settings.local.json")
}

// McpJsonPath returns <cwd>/.mcp.json.
func McpJsonPath(cwd string) string {
	return filepath.Join(cwd, ".mcp.json")
}

// SettingsGoJsonPath returns <cwd>/.harness/settings.go.json.
func SettingsGoJsonPath(cwd string) string {
	return filepath.Join(cwd, ".harness", "settings.go.json")
}

// LoadSettingsGoJson loads the full contents of .harness/settings.go.json.
// Returns nil if the file does not exist.
func LoadSettingsGoJson(cwd string) (map[string]json.RawMessage, error) {
	path := SettingsGoJsonPath(cwd)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc, nil
}

// SaveSettingsGoJson writes the full document to .harness/settings.go.json.
// Creates the .harness directory if it doesn't exist.
func SaveSettingsGoJson(cwd string, doc map[string]json.RawMessage) error {
	path := SettingsGoJsonPath(cwd)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .harness dir: %w", err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings.go.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// LoadSettingsGoJsonMcpServers loads mcpServers from .harness/settings.go.json.
// Returns nil, nil if the file or key is missing.
func LoadSettingsGoJsonMcpServers(cwd string) (map[string]json.RawMessage, error) {
	doc, err := LoadSettingsGoJson(cwd)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}
	raw, ok := doc["mcpServers"]
	if !ok {
		return nil, nil
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("parse mcpServers in settings.go.json: %w", err)
	}
	return servers, nil
}

// SaveSettingsGoJsonMcpServers writes mcpServers into .harness/settings.go.json,
// preserving all other top-level keys.
func SaveSettingsGoJsonMcpServers(cwd string, servers map[string]json.RawMessage) error {
	doc, err := LoadSettingsGoJson(cwd)
	if err != nil {
		return err
	}
	if doc == nil {
		doc = make(map[string]json.RawMessage)
	}
	raw, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("marshal mcpServers: %w", err)
	}
	doc["mcpServers"] = raw
	return SaveSettingsGoJson(cwd, doc)
}
