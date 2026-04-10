package permissionrules

import (
	"testing"

	"goc/types"
)

func TestMcpInfoFromString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want *McpInfoStringResult
	}{
		{"mcp__github__list_issues", &McpInfoStringResult{ServerName: "github", ToolName: strPtr("list_issues")}},
		{"Bash", nil},
		{"grep__pattern", nil},
		{"mcp__", nil},
		{"mcp__server", &McpInfoStringResult{ServerName: "server", ToolName: nil}},
		{"mcp__server__tool__with__underscores", &McpInfoStringResult{ServerName: "server", ToolName: strPtr("tool__with__underscores")}},
		{"", nil},
	}
	for _, tc := range cases {
		got := McpInfoFromString(tc.in)
		if !mcpInfoEqual(got, tc.want) {
			t.Fatalf("McpInfoFromString(%q): got %#v want %#v", tc.in, got, tc.want)
		}
	}
}

func mcpInfoEqual(a, b *McpInfoStringResult) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.ServerName != b.ServerName {
		return false
	}
	if (a.ToolName == nil) != (b.ToolName == nil) {
		return false
	}
	if a.ToolName != nil && b.ToolName != nil && *a.ToolName != *b.ToolName {
		return false
	}
	return true
}

func strPtr(s string) *string { return &s }

func TestGetMcpPrefix(t *testing.T) {
	t.Parallel()
	if got := GetMcpPrefix("github"); got != "mcp__github__" {
		t.Fatalf("GetMcpPrefix(github) = %q", got)
	}
	if got := GetMcpPrefix("my-server"); got != "mcp__my-server__" {
		t.Fatalf("GetMcpPrefix(my-server) = %q", got)
	}
	if got := GetMcpPrefix("my.server"); got != "mcp__my_server__" {
		t.Fatalf("GetMcpPrefix(my.server) = %q", got)
	}
}

func TestBuildMcpToolName(t *testing.T) {
	t.Parallel()
	if got := BuildMcpToolName("github", "list_issues"); got != "mcp__github__list_issues" {
		t.Fatalf("BuildMcpToolName = %q", got)
	}
	if got := BuildMcpToolName("my.server", "my.tool"); got != "mcp__my_server__my_tool" {
		t.Fatalf("BuildMcpToolName my.server = %q", got)
	}
}

func TestGetToolNameForPermissionCheck(t *testing.T) {
	t.Parallel()
	mcp := &types.MCPInfo{ServerName: "github", ToolName: "list_issues"}
	tool := types.ToolSpec{Name: "list_issues", MCPInfo: mcp}
	if got := GetToolNameForPermissionCheck(tool); got != "mcp__github__list_issues" {
		t.Fatalf("MCP tool: got %q", got)
	}
	if got := GetToolNameForPermissionCheck(types.ToolSpec{Name: "Bash"}); got != "Bash" {
		t.Fatalf("Bash: got %q", got)
	}
	if got := GetToolNameForPermissionCheck(types.ToolSpec{Name: "Write", MCPInfo: nil}); got != "Write" {
		t.Fatalf("Write: got %q", got)
	}
}
