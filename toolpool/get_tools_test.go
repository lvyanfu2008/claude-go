package toolpool

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestParseToolsAPIDocumentJSON_minimal(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"tools":[{"name":"Alpha","description":"d","input_schema":{"type":"object"}}]}`)
	got, err := ParseToolsAPIDocumentJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Alpha" || got[0].Description != "d" {
		t.Fatalf("%+v", got)
	}
}

func TestGetTools_removesSpecialIfPresent(t *testing.T) {
	base := []types.ToolSpec{
		{Name: "Read"},
		{Name: ListMcpResourcesToolName},
		{Name: "Bash"},
	}
	ctx := types.EmptyToolPermissionContextData()
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	out := GetTools(ctx, base)
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
}

func TestGetTools_simpleMode(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "1")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	base := []types.ToolSpec{
		{Name: "Bash"}, {Name: "Read"}, {Name: "Edit"}, {Name: "Agent"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	if len(out) != 3 {
		t.Fatalf("want 3 tools got %#v", out)
	}
}

func TestMarshalToolsAPIDocumentDefinitions(t *testing.T) {
	t.Parallel()
	raw, err := MarshalToolsAPIDocumentDefinitions([]types.ToolSpec{
		{Name: "Bash", Description: "x", InputJSONSchema: json.RawMessage(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var a []map[string]any
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0]["name"] != "Bash" {
		t.Fatalf("%s", raw)
	}
}

func TestToolSpecsFromEmbeddedToolsAPIJSON(t *testing.T) {
	t.Parallel()
	specs, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) < 5 {
		t.Fatalf("embedded tools_api.json too small: %d", len(specs))
	}
}
