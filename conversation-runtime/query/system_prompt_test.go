package query

import (
	"strings"
	"testing"
)

func TestSplitSysPromptPrefix_WithBoundaryMarker(t *testing.T) {
	sp := SystemPrompt{
		"x-anthropic-billing-header: cc_version=1.0.0; cc_entrypoint=cli;",
		cliSyspromptPrefix,
		"# Static section 1\ncontent here",
		"# Static section 2\nmore content",
		systemPromptDynamicBoundary,
		"# Dynamic section 1\nsession-specific guidance",
		"# Dynamic section 2\nenvironment info",
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}

	// Block 0: attribution header — no cache
	if blocks[0].CacheScope != nil {
		t.Errorf("block 0 (attribution): expected nil CacheScope, got %v", *blocks[0].CacheScope)
	}
	if !strings.HasPrefix(blocks[0].Text, "x-anthropic-billing-header") {
		t.Errorf("block 0 should be attribution header, got: %s", blocks[0].Text[:50])
	}

	// Block 1: CLI prefix — no cache
	if blocks[1].CacheScope != nil {
		t.Errorf("block 1 (prefix): expected nil CacheScope, got %v", *blocks[1].CacheScope)
	}
	if !strings.Contains(blocks[1].Text, "Harness Code") {
		t.Errorf("block 1 should be CLI prefix, got: %s", blocks[1].Text[:50])
	}

	// Block 2: static content — global cache
	if blocks[2].CacheScope == nil {
		t.Error("block 2 (static): expected non-nil CacheScope")
	} else if *blocks[2].CacheScope != CacheScopeGlobal {
		t.Errorf("block 2 (static): expected CacheScopeGlobal, got %v", *blocks[2].CacheScope)
	}
	if !strings.Contains(blocks[2].Text, "Static section 1") {
		t.Errorf("block 2 should contain static content, got: %s", blocks[2].Text[:50])
	}

	// Block 3: dynamic content — nil cache
	if len(blocks) > 3 {
		if blocks[3].CacheScope != nil {
			t.Errorf("block 3 (dynamic): expected nil CacheScope, got %v", *blocks[3].CacheScope)
		}
		if !strings.Contains(blocks[3].Text, "Dynamic section 1") {
			t.Errorf("block 3 should contain dynamic content, got: %s", blocks[3].Text[:50])
		}
	}
}

func TestSplitSysPromptPrefix_NoBoundaryMarker(t *testing.T) {
	sp := SystemPrompt{
		cliSyspromptPrefix,
		"# Some content",
		"# More content",
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (prefix + rest), got %d", len(blocks))
	}

	// Block 0: prefix — org cache (fallback mode)
	if blocks[0].CacheScope == nil {
		t.Error("block 0 (prefix, no boundary): expected non-nil CacheScope")
	} else if *blocks[0].CacheScope != CacheScopeOrg {
		t.Errorf("block 0 (prefix): expected CacheScopeOrg, got %v", *blocks[0].CacheScope)
	}

	// Block 1: rest — org cache
	if blocks[1].CacheScope == nil {
		t.Error("block 1 (rest, no boundary): expected non-nil CacheScope")
	} else if *blocks[1].CacheScope != CacheScopeOrg {
		t.Errorf("block 1 (rest): expected CacheScopeOrg, got %v", *blocks[1].CacheScope)
	}
}

func TestSplitSysPromptPrefix_AttributionHeaderOnly(t *testing.T) {
	sp := SystemPrompt{
		"x-anthropic-billing-header: cc_version=1.0.0; cc_entrypoint=cli;",
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].CacheScope != nil {
		t.Errorf("block 0 (attribution): expected nil CacheScope, got %v", *blocks[0].CacheScope)
	}
	if !strings.HasPrefix(blocks[0].Text, "x-anthropic-billing-header") {
		t.Errorf("block 0 should be attribution header")
	}
}

func TestSplitSysPromptPrefix_EmptyInput(t *testing.T) {
	blocks := SplitSysPromptPrefix(nil)
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for nil input, got %d", len(blocks))
	}

	blocks = SplitSysPromptPrefix(SystemPrompt{})
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for empty input, got %d", len(blocks))
	}
}

func TestSplitSysPromptPrefix_OnlyBoundaryMarker(t *testing.T) {
	sp := SystemPrompt{
		systemPromptDynamicBoundary,
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for only-boundary input, got %d", len(blocks))
	}
}

func TestSplitSysPromptPrefix_BoundaryAtStart(t *testing.T) {
	// All content is after boundary — all dynamic, no cache
	sp := SystemPrompt{
		systemPromptDynamicBoundary,
		"# Dynamic only",
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].CacheScope != nil {
		t.Errorf("block 0 (all dynamic): expected nil CacheScope, got %v", *blocks[0].CacheScope)
	}
}

func TestSplitSysPromptPrefix_BoundaryAtEnd(t *testing.T) {
	// All content is before boundary — all static, global cache
	sp := SystemPrompt{
		"# Static only",
		systemPromptDynamicBoundary,
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].CacheScope == nil {
		t.Error("block 0 (all static): expected non-nil CacheScope")
	} else if *blocks[0].CacheScope != CacheScopeGlobal {
		t.Errorf("block 0 (all static): expected CacheScopeGlobal, got %v", *blocks[0].CacheScope)
	}
}

func TestGetCLISyspromptPrefix(t *testing.T) {
	prefix := GetCLISyspromptPrefix()
	if !strings.Contains(prefix, "Harness Code") {
		t.Errorf("unexpected prefix: %s", prefix)
	}
}

func TestGetAttributionHeader_Disabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_ATTRIBUTION_HEADER", "1")
	h := GetAttributionHeader()
	if h != "" {
		t.Errorf("expected empty attribution when disabled, got: %s", h)
	}
}

func TestGetAttributionHeader_CustomVersion(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_ATTRIBUTION_HEADER", "")
	t.Setenv("CLAUDE_CODE_VERSION", "2.1.888")
	h := GetAttributionHeader()
	if !strings.Contains(h, "cc_version=2.1.888") {
		t.Errorf("expected version in header, got: %s", h)
	}
	if !strings.HasPrefix(h, "x-anthropic-billing-header:") {
		t.Errorf("expected attribution header prefix, got: %s", h)
	}
}

func TestHasAdvisorModel(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ADVISOR_MODEL", "")
	if HasAdvisorModel() {
		t.Error("expected no advisor model when env is empty")
	}

	t.Setenv("CLAUDE_CODE_ADVISOR_MODEL", "claude-sonnet-4-6")
	if !HasAdvisorModel() {
		t.Error("expected advisor model to be detected")
	}
}

func TestHasChromeTools(t *testing.T) {
	if HasChromeTools(nil) {
		t.Error("expected false for nil tools")
	}
	if HasChromeTools([]byte(`[]`)) {
		t.Error("expected false for empty tools")
	}
	if !HasChromeTools([]byte(`[{"name":"mcp__claude-in-chrome__tabs_context_mcp"}]`)) {
		t.Error("expected true for chrome tools with mcp__claude-in-chrome__ prefix")
	}
	// False: no chrome tools, just regular tools
	if HasChromeTools([]byte(`[{"name":"BashTool"},{"name":"ReadTool"}]`)) {
		t.Error("expected false for non-chrome tools")
	}
	// Partial substring should match too
	if !HasChromeTools([]byte(`[{"name":"BashTool"},{"name":"mcp__claude-in-chrome__read_console_messages"}]`)) {
		t.Error("expected true for mixed tools with chrome")
	}
}

func TestAdvisorInstructionsNotEmpty(t *testing.T) {
	if ADVISOR_TOOL_INSTRUCTIONS == "" {
		t.Error("ADVISOR_TOOL_INSTRUCTIONS must not be empty")
	}
	if !strings.Contains(ADVISOR_TOOL_INSTRUCTIONS, "advisor") {
		t.Error("ADVISOR_TOOL_INSTRUCTIONS should contain 'advisor'")
	}
}

func TestChromeInstructionsNotEmpty(t *testing.T) {
	if CHROME_TOOL_SEARCH_INSTRUCTIONS == "" {
		t.Error("CHROME_TOOL_SEARCH_INSTRUCTIONS must not be empty")
	}
	if !strings.Contains(CHROME_TOOL_SEARCH_INSTRUCTIONS, "claude-in-chrome") {
		t.Error("CHROME_TOOL_SEARCH_INSTRUCTIONS should contain 'claude-in-chrome'")
	}
}

func TestSplitSysPromptPrefix_WithAdvisorAndChrome(t *testing.T) {
	sp := SystemPrompt{
		"x-anthropic-billing-header: cc_version=1.0.0; cc_entrypoint=cli;",
		cliSyspromptPrefix,
		systemPromptDynamicBoundary,
		"# Dynamic content",
		ADVISOR_TOOL_INSTRUCTIONS,
		CHROME_TOOL_SEARCH_INSTRUCTIONS,
	}

	blocks := SplitSysPromptPrefix(sp)
	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}

	// Advisor and Chrome instructions are after the boundary, so they should be dynamic (no cache)
	dynamicBlock := blocks[len(blocks)-1]
	if dynamicBlock.CacheScope != nil {
		t.Errorf("advisor+chrome block should be dynamic (nil cache), got %v", *dynamicBlock.CacheScope)
	}
	if !strings.Contains(dynamicBlock.Text, "advisor") {
		t.Error("dynamic block should contain advisor instructions")
	}
	if !strings.Contains(dynamicBlock.Text, "claude-in-chrome") {
		t.Error("dynamic block should contain chrome instructions")
	}
}
