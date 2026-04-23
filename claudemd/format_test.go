package claudemd

import (
	"os"
	"testing"
)

func TestFilterInjectedMemoryFilesDefaultBehavior(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv("CLAUDE_CODE_TENGU_MOTH_COPSE")
	os.Unsetenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")

	testFiles := []MemoryFileInfo{
		{Type: MemoryAutoMem, Path: "auto/MEMORY.md"},
		{Type: MemoryTeamMem, Path: "team/MEMORY.md"},
		{Type: MemoryUser, Path: "user.md"},
		{Type: MemoryProject, Path: "project.md"},
	}

	// Test default behavior (should filter out MEMORY.md files)
	filtered := FilterInjectedMemoryFiles(testFiles)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files after filtering, got %d", len(filtered))
	}
	
	// Verify only user and project files remain
	for _, f := range filtered {
		if f.Type == MemoryAutoMem || f.Type == MemoryTeamMem {
			t.Errorf("Should not include memory index files in default behavior, but found %s", f.Type)
		}
	}

	// Test explicit disable of filtering
	os.Setenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX", "1")
	filtered = FilterInjectedMemoryFiles(testFiles)
	if len(filtered) != 4 {
		t.Errorf("When filtering disabled, expected 4 files, got %d", len(filtered))
	}

	// Clean up
	os.Unsetenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")
}