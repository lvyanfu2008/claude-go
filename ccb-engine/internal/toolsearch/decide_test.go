package toolsearch

import (
	"strings"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func TestBuildWireConfig_tstAuto_respectsCharThreshold(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:10")
	t.Setenv("CLAUDE_CODE_GO_TST_AUTO_CHAR_SCALE", "1")
	// Tiny deferred payload: below default 10% of 200k * 2.5 chars threshold.
	small := []anthropic.ToolDefinition{
		{Name: ToolSearchToolName, Description: "x", InputSchema: map[string]any{"type": "object"}},
		{Name: "TodoWrite", Description: "y", InputSchema: map[string]any{"type": "object"}},
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", small, false, false)
	if cfg.UseDynamicToolLoading {
		t.Fatal("expected tst-auto below threshold → standard path")
	}
}

func TestBuildWireConfig_tstAuto_defaultScaleCrossesNearExportSize(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:10")
	t.Setenv("CLAUDE_CODE_GO_TST_AUTO_CHAR_SCALE", "")
	// ~32k raw deferred chars * default scale (~1.65) crosses ~50k threshold (tools_api.json ballpark).
	pad := strings.Repeat("x", 31800)
	tools := []anthropic.ToolDefinition{
		{Name: ToolSearchToolName, Description: "s", InputSchema: map[string]any{"type": "object"}},
		{Name: "TodoWrite", Description: pad, InputSchema: map[string]any{"type": "object"}},
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	if !cfg.UseDynamicToolLoading {
		t.Fatal("expected tst-auto with default scale to enable dynamic for export-sized static descriptions")
	}
}

func TestBuildWireConfig_pendingMcpKeepsDynamicWithNoBuiltinDeferred(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	onlySearch := []anthropic.ToolDefinition{
		{Name: ToolSearchToolName, Description: "s", InputSchema: map[string]any{"type": "object"}},
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", onlySearch, true, false)
	if !cfg.UseDynamicToolLoading {
		t.Fatal("expected pending MCP to keep dynamic loading on")
	}
}

func TestStripToolReferences_placeholder(t *testing.T) {
	m := anthropic.Message{
		Role: "user",
		Content: []anthropic.ContentBlock{
			{Type: "tool_result", Content: []any{map[string]any{"type": "tool_reference", "tool_name": "X"}}},
		},
	}
	out := stripToolReferencesFromUser(m)
	blocks := out.Content.([]anthropic.ContentBlock)
	inner := blocks[0].Content.([]any)
	if len(inner) != 1 {
		t.Fatalf("got %#v", inner)
	}
	tm, _ := inner[0].(map[string]any)
	if tm["type"] != "text" {
		t.Fatalf("want text placeholder, got %#v", tm)
	}
}
