package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildMcpAuthToolName(t *testing.T) {
	got := BuildMcpAuthToolName("github")
	want := "mcp__github__authenticate"
	if got != want {
		t.Fatalf("BuildMcpAuthToolName: got %q, want %q", got, want)
	}
}

func TestParseServerFromMcpAuthName(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		{"mcp__github__authenticate", "github"},
		{"mcp__filesystem__authenticate", "filesystem"},
		{"mcp__slack__authenticate", "slack"},
		{"Agent", ""},
		{"mcp__github__read_file", ""},
		{"", ""},
		{"mcp____authenticate", ""},
	}
	for _, tt := range tests {
		got := parseServerFromMcpAuthName(tt.toolName)
		if got != tt.want {
			t.Errorf("parseServerFromMcpAuthName(%q) = %q, want %q", tt.toolName, got, tt.want)
		}
	}
}

func TestCreateMcpAuthToolSpec(t *testing.T) {
	spec := CreateMcpAuthToolSpec("github", "Authenticate with GitHub MCP server")
	if spec.Name != "mcp__github__authenticate" {
		t.Fatalf("Name: got %q, want %q", spec.Name, "mcp__github__authenticate")
	}
	if spec.IsMcp == nil || !*spec.IsMcp {
		t.Fatal("IsMcp should be true")
	}
	if spec.MCPInfo == nil || spec.MCPInfo.ToolName != "authenticate" || spec.MCPInfo.ServerName != "github" {
		t.Fatal("MCPInfo not set correctly")
	}
	if spec.Description != "Authenticate with GitHub MCP server" {
		t.Fatalf("Description: got %q", spec.Description)
	}
}

func TestMcpAuthFromJSONNoHandler(t *testing.T) {
	// Clear handler
	old := McpAuthToolHandler
	McpAuthToolHandler = nil
	defer func() { McpAuthToolHandler = old }()

	result, _, err := McpAuthFromJSON(context.Background(), "mcp__github__authenticate", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Data McpAuthCallResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid result: %v", err)
	}
	if out.Data.Status != "error" {
		t.Fatalf("expected error status, got %q", out.Data.Status)
	}
	if !strings.Contains(out.Data.Message, "not configured") {
		t.Fatalf("expected 'not configured' message, got %q", out.Data.Message)
	}
}

func TestMcpAuthFromJSONBadToolName(t *testing.T) {
	result, _, err := McpAuthFromJSON(context.Background(), "Agent", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Data McpAuthCallResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid result: %v", err)
	}
	if out.Data.Status != "error" {
		t.Fatalf("expected error status, got %q", out.Data.Status)
	}
	if !strings.Contains(out.Data.Message, "parse") {
		t.Fatalf("expected parse error, got %q", out.Data.Message)
	}
}

func TestMcpAuthFromJSONWithHandler(t *testing.T) {
	old := McpAuthToolHandler
	McpAuthToolHandler = func(_ context.Context, serverName string) (map[string]any, error) {
		if serverName != "github" {
			t.Fatalf("expected serverName 'github', got %q", serverName)
		}
		return map[string]any{
			"status":  "auth_url",
			"authUrl": "https://github.com/login/oauth/authorize",
			"message": "Open the URL to authenticate.",
		}, nil
	}
	defer func() { McpAuthToolHandler = old }()

	result, _, err := McpAuthFromJSON(context.Background(), "mcp__github__authenticate", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid result: %v", err)
	}
	if out.Data["status"] != "auth_url" {
		t.Fatalf("expected auth_url status, got %v", out.Data["status"])
	}
}

func TestMcpAuthFromJSONHandlerError(t *testing.T) {
	old := McpAuthToolHandler
	McpAuthToolHandler = func(_ context.Context, _ string) (map[string]any, error) {
		return nil, context.DeadlineExceeded
	}
	defer func() { McpAuthToolHandler = old }()

	result, _, err := McpAuthFromJSON(context.Background(), "mcp__github__authenticate", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Data McpAuthCallResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid result: %v", err)
	}
	if out.Data.Status != "error" {
		t.Fatalf("expected error status, got %q", out.Data.Status)
	}
}
