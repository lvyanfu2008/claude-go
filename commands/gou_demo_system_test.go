package commands

import (
	"os"
	"testing"
)

func TestMemorySkipIndexDefaultBehavior(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv("FEATURE_MOTH_COPSE")
	os.Unsetenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")
	
	// Test default behavior (should be true to match TS)
	var opts GouDemoSystemOpts
	ApplyGouDemoRuntimeEnv(&opts)
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