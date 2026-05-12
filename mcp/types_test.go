package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseMcpStdioServerConfig(t *testing.T) {
	data := []byte(`{"command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]}`)
	cfg, err := ParseMcpServerConfig(data)
	if err != nil {
		t.Fatalf("ParseMcpServerConfig failed: %v", err)
	}
	stdio, ok := cfg.(McpStdioServerConfig)
	if !ok {
		t.Fatalf("expected McpStdioServerConfig, got %T", cfg)
	}
	if stdio.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", stdio.Command)
	}
	if len(stdio.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(stdio.Args))
	}
}

func TestParseMcpSSEServerConfig(t *testing.T) {
	data := []byte(`{"type": "sse", "url": "https://example.com/sse"}`)
	cfg, err := ParseMcpServerConfig(data)
	if err != nil {
		t.Fatalf("ParseMcpServerConfig failed: %v", err)
	}
	sse, ok := cfg.(McpSSEServerConfig)
	if !ok {
		t.Fatalf("expected McpSSEServerConfig, got %T", cfg)
	}
	if sse.URL != "https://example.com/sse" {
		t.Errorf("expected URL, got %q", sse.URL)
	}
}

func TestParseMcpHTTPServerConfig(t *testing.T) {
	data := []byte(`{"type": "http", "url": "https://example.com/mcp", "headers": {"Authorization": "Bearer test"}}`)
	cfg, err := ParseMcpServerConfig(data)
	if err != nil {
		t.Fatalf("ParseMcpServerConfig failed: %v", err)
	}
	httpCfg, ok := cfg.(McpHTTPServerConfig)
	if !ok {
		t.Fatalf("expected McpHTTPServerConfig, got %T", cfg)
	}
	if httpCfg.URL != "https://example.com/mcp" {
		t.Errorf("expected URL, got %q", httpCfg.URL)
	}
}

func TestParseMcpJsonConfig(t *testing.T) {
	data := []byte(`{"mcpServers": {"filesystem": {"command": "npx", "args": ["-y", "test"]}}}`)
	var cfg McpJsonConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal McpJsonConfig: %v", err)
	}
	if len(cfg.McpServers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.McpServers))
	}
	raw, ok := cfg.McpServers["filesystem"]
	if !ok {
		t.Fatal("missing 'filesystem' server")
	}
	_, err := ParseMcpServerConfig(raw)
	if err != nil {
		t.Fatalf("parse 'filesystem' config: %v", err)
	}
}

func TestNormalizeMcpServerName(t *testing.T) {
	tests := []struct{ input, want string }{
		{"My Server", "my_server"},
		{"test-server", "test-server"},
		{"UPPER_CASE", "upper_case"},
		{"spaces   here", "spaces___here"},
	}
	for _, tt := range tests {
		got := NormalizeMcpServerName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeMcpServerName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildMcpToolName(t *testing.T) {
	name := BuildMcpToolName("filesystem", "read_file")
	if name != "mcp__filesystem__read_file" {
		t.Errorf("expected mcp__filesystem__read_file, got %q", name)
	}
}

func TestIsMcpTool(t *testing.T) {
	if !IsMcpTool("mcp__filesystem__read_file") {
		t.Error("expected true for mcp__ prefix")
	}
	if IsMcpTool("BashTool") {
		t.Error("expected false for non-mcp tool")
	}
}

func TestParseMcpToolName(t *testing.T) {
	server, tool, ok := ParseMcpToolName("mcp__filesystem__read_file")
	if !ok {
		t.Fatal("ParseMcpToolName failed")
	}
	if server != "filesystem" || tool != "read_file" {
		t.Errorf("expected filesystem/read_file, got %s/%s", server, tool)
	}

	_, _, ok = ParseMcpToolName("BashTool")
	if ok {
		t.Error("expected false for non-mcp name")
	}
}

func TestAddScopeToServers(t *testing.T) {
	servers := map[string]McpServerConfig{
		"test": McpStdioServerConfig{Command: "echo"},
	}
	result := AddScopeToServers(servers, ScopeUser)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	scoped, ok := result["test"]
	if !ok || scoped.Scope != ScopeUser {
		t.Error("scope not set correctly")
	}
}
