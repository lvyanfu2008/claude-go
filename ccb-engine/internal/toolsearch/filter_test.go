package toolsearch

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func sampleToolsAPIStyle() []anthropic.ToolDefinition {
	raw := []byte(`[
	  {"name":"Agent","description":"a","input_schema":{"type":"object"}},
	  {"name":"ToolSearch","description":"s","input_schema":{"type":"object"}},
	  {"name":"TodoWrite","description":"t","input_schema":{"type":"object"}},
	  {"name":"Read","description":"r","input_schema":{"type":"object"}}
	]`)
	var out []anthropic.ToolDefinition
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func TestApplyWire_openAICompat_dynamic_excludesDeferredUntilDiscovered(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	tools := sampleToolsAPIStyle()
	cfg := BuildWireConfig("deepseek-chat", tools, false, true)
	if !cfg.UseDynamicToolLoading || !cfg.OpenAICompat {
		t.Fatalf("cfg=%+v", cfg)
	}
	got := ApplyWire(tools, nil, cfg)
	names := toolNames(got)
	want := map[string]bool{"Agent": true, "ToolSearch": true, "Read": true}
	if len(names) != len(want) {
		t.Fatalf("got %v", names)
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected %q", n)
		}
	}
}

func TestApplyWire_dynamic_excludesDeferredUntilDiscovered(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	tools := sampleToolsAPIStyle()
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	got := ApplyWire(tools, nil, cfg)
	names := toolNames(got)
	if len(names) != 3 {
		t.Fatalf("got %v (len=%d) want 3 tools first turn", names, len(names))
	}
	want := map[string]bool{"Agent": true, "ToolSearch": true, "Read": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected tool %q", n)
		}
	}
	for _, tdef := range got {
		if tdef.Name == "TodoWrite" {
			t.Fatal("TodoWrite should not appear before discovery")
		}
	}
}

func TestApplyWire_afterToolReference_includesDeferredWithDeferLoading(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	tools := sampleToolsAPIStyle()
	msgs := []anthropic.Message{
		{
			Role: "user",
			Content: []anthropic.ContentBlock{
				{
					Type: "tool_result",
					Content: []any{
						map[string]any{"type": "tool_reference", "tool_name": "TodoWrite"},
					},
				},
			},
		},
	}
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	got := ApplyWire(tools, msgs, cfg)
	var todo *anthropic.ToolDefinition
	for i := range got {
		if got[i].Name == "TodoWrite" {
			todo = &got[i]
			break
		}
	}
	if todo == nil {
		t.Fatal("expected TodoWrite after discovery")
	}
	if todo.DeferLoading == nil || !*todo.DeferLoading {
		t.Fatalf("TodoWrite defer_loading=%v want true", todo.DeferLoading)
	}
}

func TestApplyWire_standard_stripsToolSearch(t *testing.T) {
	t.Setenv("ENABLE_TOOL_SEARCH", "false")
	tools := sampleToolsAPIStyle()
	cfg := BuildWireConfig("claude-sonnet-4-20250514", tools, false, false)
	got := ApplyWire(tools, nil, cfg)
	names := toolNames(got)
	for _, n := range names {
		if n == "ToolSearch" {
			t.Fatal("ToolSearch should be stripped in standard mode")
		}
	}
	if len(names) != 3 {
		t.Fatalf("got %v want Agent,Read,TodoWrite", names)
	}
}

func TestApplyWire_haiku_disablesDynamic(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	tools := sampleToolsAPIStyle()
	cfg := BuildWireConfig("claude-3-5-haiku-20241022", tools, false, false)
	got := ApplyWire(tools, nil, cfg)
	for _, n := range toolNames(got) {
		if n == "ToolSearch" {
			t.Fatal("ToolSearch should be stripped when model does not support tool_reference")
		}
	}
}

func TestExtractDiscovered_compactBoundary(t *testing.T) {
	meta, _ := json.Marshal(map[string]any{"preCompactDiscoveredTools": []string{"TodoWrite", "WebFetch"}})
	msgs := []anthropic.Message{
		{Type: "system", Subtype: "compact_boundary", CompactMetadata: meta},
	}
	got := ExtractDiscoveredToolNames(msgs)
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
}

func TestPrepareAnthropicMessages_prependDeferredList(t *testing.T) {
	t.Setenv("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA", "")
	t.Setenv("CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT", "1")
	tools := sampleToolsAPIStyle()
	cfg := WireConfig{
		UseDynamicToolLoading:         true,
		ModelSupportsToolReference:    true,
		PrependAvailableDeferredBlock: true,
	}
	out := PrepareAnthropicMessages([]anthropic.Message{{Role: "user", Content: "hi"}}, tools, cfg)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	s, ok := out[0].Content.(string)
	if !ok || !strings.Contains(s, "<available-deferred-tools>") {
		t.Fatalf("first message should be deferred list, got %#v", out[0].Content)
	}
}

func toolNames(tools []anthropic.ToolDefinition) []string {
	var s []string
	for _, t := range tools {
		s = append(s, t.Name)
	}
	return s
}

func TestBetasForWiredTools(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	tools := []anthropic.ToolDefinition{{Name: ToolSearchToolName, InputSchema: map[string]any{}}}
	b := BetasForWiredTools(tools)
	if len(b) != 1 || b[0] == "" {
		t.Fatalf("betas=%v", b)
	}
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	if got := BetasForWiredTools(tools); len(got) != 0 {
		t.Fatalf("expected no betas when kill switch on, got %v", got)
	}
}
