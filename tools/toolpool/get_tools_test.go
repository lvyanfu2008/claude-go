package toolpool

import (
	"testing"

	"goc/types"
)

func TestGetToolsToolSearchComesFromBaseRegistryGate(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "0")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")

	base := ToolSpecsFromGoWire()
	out := GetTools(types.EmptyToolPermissionContextData(), base)
	for _, tool := range out {
		if tool.Name == "ToolSearch" {
			t.Fatalf("ToolSearch should be absent when registry gate is off")
		}
	}
}

func TestGetToolsToolSearchPresentWhenRegistryGateOn(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")

	base := ToolSpecsFromGoWire()
	out := GetTools(types.EmptyToolPermissionContextData(), base)
	found := false
	for _, tool := range out {
		if tool.Name == "ToolSearch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ToolSearch should be present when registry gate is on")
	}
}

func TestGetToolsFiltersSpecialToolsFromModelFacingList(t *testing.T) {
	exportedBase := []types.ToolSpec{
		{Name: "Read"},
		{Name: "ListMcpResourcesTool"},
		{Name: "ReadMcpResourceTool"},
		{Name: "StructuredOutput"},
	}
	out := GetTools(types.EmptyToolPermissionContextData(), exportedBase)
	for _, tool := range out {
		if tool.Name == "ListMcpResourcesTool" || tool.Name == "ReadMcpResourceTool" || tool.Name == "StructuredOutput" {
			t.Fatalf("special tool %q should be filtered from model-facing list", tool.Name)
		}
	}
	if len(out) != 1 || out[0].Name != "Read" {
		t.Fatalf("expected only Read to remain, got=%v", toolNamesGT(out))
	}
}

func toolNamesGT(specs []types.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}
