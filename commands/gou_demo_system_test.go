package commands

import (
	"os"
	"testing"
)

func TestMemorySkipIndexDefaultBehavior(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv("FEATURE_MOTH_COPSE")
	os.Unsetenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")
	os.Unsetenv("CLAUDE_CODE_GO_KAIROS_ACTIVE")
	
	// Test default behavior (should be false to match TS STATE.kairosActive default)
	var opts GouDemoSystemOpts
	ApplyGouDemoRuntimeEnv(&opts)
	if opts.KairosActive {
		t.Error("KairosActive should default to false when CLAUDE_CODE_GO_KAIROS_ACTIVE is unset, matching TS STATE.kairosActive default")
	}
	if !opts.MemorySkipIndex {
		t.Error("MemorySkipIndex should default to true to match TypeScript behavior")
	}
	
	// Test explicit disable
	os.Setenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX", "1")
	opts = GouDemoSystemOpts{}
	ApplyGouDemoRuntimeEnv(&opts)
	if opts.MemorySkipIndex {
		t.Error("MemorySkipIndex should be false when explicitly disabled")
	}
	
	// Clean up
	os.Unsetenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")
}