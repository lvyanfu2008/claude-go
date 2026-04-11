package toolsearch

import (
	"encoding/json"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func TestEffectiveToolSearchMode_embeddedPromotesTstAutoToTst(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:50")
	t.Setenv("GOU_DEMO_USE_EMBEDDED_TOOLS_API", "1")
	raw := []byte(`[
	  {"name":"ToolSearch","description":"s","input_schema":{"type":"object"}},
	  {"name":"TodoWrite","description":"t","input_schema":{"type":"object"}}
	]`)
	var tools []anthropic.ToolDefinition
	if err := json.Unmarshal(raw, &tools); err != nil {
		t.Fatal(err)
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	if !cfg.UseDynamicToolLoading {
		t.Fatal("embedded tools API should force tst-auto→tst so deferred tools are not all inlined in tools[]")
	}
}

func TestToolSearchContextZero_skipsPrependOnly(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	t.Setenv("CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT", "0")
	t.Setenv("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA", "")
	raw := []byte(`[
	  {"name":"ToolSearch","description":"s","input_schema":{"type":"object"}},
	  {"name":"TodoWrite","description":"t","input_schema":{"type":"object"}}
	]`)
	var tools []anthropic.ToolDefinition
	if err := json.Unmarshal(raw, &tools); err != nil {
		t.Fatal(err)
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	if !cfg.UseDynamicToolLoading {
		t.Fatal("dynamic tools[] filtering should stay on")
	}
	if cfg.PrependAvailableDeferredBlock {
		t.Fatal("prepend should be off when CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT=0")
	}
}
