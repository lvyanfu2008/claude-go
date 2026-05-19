package querycontext

import (
	"strings"
	"testing"
)

func TestGetCoordinatorUserContext_NotActive(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")

	result := GetCoordinatorUserContext(nil, "")
	if result != nil {
		t.Errorf("expected nil when coordinator mode not active, got %v", result)
	}
}

func TestGetCoordinatorUserContext_Active(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")

	result := GetCoordinatorUserContext(nil, "")
	if result == nil {
		t.Fatal("expected non-nil result when coordinator mode active")
	}

	ctx, ok := result["workerToolsContext"]
	if !ok {
		t.Fatal("expected workerToolsContext key")
	}

	if !strings.Contains(ctx, "Workers spawned via the Agent tool have access to these tools:") {
		t.Errorf("expected worker tools intro, got: %s", ctx[:100])
	}

	// Should contain all 20 tools
	for _, tool := range workerToolsAll {
		if !strings.Contains(ctx, tool) {
			t.Errorf("expected tool %q in worker context", tool)
		}
	}

	// Should NOT contain internal worker tools
	for _, excluded := range []string{"SendMessage", "TeamCreate", "TeamDelete", "SyntheticOutput"} {
		if strings.Contains(ctx, excluded+",") || strings.Contains(ctx, excluded+":") {
			t.Errorf("tool %q should NOT be in worker context", excluded)
		}
	}
}

func TestGetCoordinatorUserContext_SimpleMode(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "1")

	result := GetCoordinatorUserContext(nil, "")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	ctx := result["workerToolsContext"]
	if !strings.Contains(ctx, "Bash, Edit, Read") {
		t.Errorf("expected simple tools list, got: %s", ctx[:100])
	}
}

func TestGetCoordinatorUserContext_WithMcpClients(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")

	result := GetCoordinatorUserContext([]string{"github", "filesystem"}, "")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	ctx := result["workerToolsContext"]
	if !strings.Contains(ctx, "MCP tools from connected MCP servers:") {
		t.Error("expected MCP server mention")
	}
	if !strings.Contains(ctx, "filesystem") && !strings.Contains(ctx, "github") {
		t.Error("expected MCP server names")
	}
}

func TestGetCoordinatorUserContext_WithScratchpad(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")

	result := GetCoordinatorUserContext(nil, "/tmp/scratch")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	ctx := result["workerToolsContext"]
	if !strings.Contains(ctx, "Scratchpad directory: /tmp/scratch") {
		t.Error("expected scratchpad directory")
	}
	if !strings.Contains(ctx, "Workers can read and write here") {
		t.Error("expected scratchpad description")
	}
}

func TestGetCoordinatorUserContext_FeatureOnly(t *testing.T) {
	// FEATURE_COORDINATOR_MODE=1 but CLAUDE_CODE_COORDINATOR_MODE is not set
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")

	result := GetCoordinatorUserContext(nil, "")
	if result != nil {
		t.Errorf("expected nil when only feature gate is on, got %v", result)
	}
}

func TestGetCoordinatorUserContext_EnvOnly(t *testing.T) {
	// CLAUDE_CODE_COORDINATOR_MODE=1 but FEATURE_COORDINATOR_MODE is not set
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")

	result := GetCoordinatorUserContext(nil, "")
	if result != nil {
		t.Errorf("expected nil when only env is on, got %v", result)
	}
}

func TestWorkerToolsAll_IsSorted(t *testing.T) {
	for i := 1; i < len(workerToolsAll); i++ {
		if workerToolsAll[i-1] >= workerToolsAll[i] {
			t.Errorf("workerToolsAll not sorted at index %d: %q >= %q", i, workerToolsAll[i-1], workerToolsAll[i])
		}
	}
}

func TestWorkerToolsAll_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, name := range workerToolsAll {
		if seen[name] {
			t.Errorf("duplicate tool name: %q", name)
		}
		seen[name] = true
	}
}
