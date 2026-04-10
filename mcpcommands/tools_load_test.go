package mcpcommands

import (
	"testing"

	"goc/permissionrules"
)

func TestParseMCPToolsJSON_byServerToolName(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"serverName":"demo","toolName":"echo","description":"d","input_schema":{"type":"object"}}]`)
	out, err := ParseMCPToolsJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("%#v", out)
	}
	want := permissionrules.BuildMcpToolName("demo", "echo")
	if out[0].Name != want {
		t.Fatalf("name %q want %q", out[0].Name, want)
	}
	if out[0].MCPInfo == nil || out[0].MCPInfo.ServerName != "demo" {
		t.Fatalf("mcpInfo %#v", out[0].MCPInfo)
	}
}

func TestParseMCPToolsJSON_byFullName(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"name":"mcp__demo__echo","input_schema":{}}]`)
	out, err := ParseMCPToolsJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "mcp__demo__echo" {
		t.Fatalf("%#v", out)
	}
}

func TestParseMCPToolsJSON_rejectNonMcpName(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"name":"Bash","input_schema":{}}]`)
	_, err := ParseMCPToolsJSON(raw)
	if err == nil {
		t.Fatal("expected error")
	}
}
