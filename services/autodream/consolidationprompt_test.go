package autodream

import (
	"strings"
	"testing"
)

func TestBuildConsolidationPrompt_basic(t *testing.T) {
	memoryRoot := "/fake/mem"
	transcriptDir := "/fake/transcripts"
	extra := "Sessions since last consolidation (2):\n- abc-123\n- def-456"

	prompt := BuildConsolidationPrompt(memoryRoot, transcriptDir, extra)

	if !strings.Contains(prompt, memoryRoot) {
		t.Errorf("expected prompt to contain memory root %q", memoryRoot)
	}
	if !strings.Contains(prompt, transcriptDir) {
		t.Errorf("expected prompt to contain transcript dir %q", transcriptDir)
	}
	if !strings.Contains(prompt, "abc-123") {
		t.Errorf("expected prompt to contain session ID")
	}
	if !strings.Contains(prompt, entrypointName) {
		t.Errorf("expected prompt to mention %s", entrypointName)
	}
	if !strings.Contains(prompt, "Phase 1") {
		t.Errorf("expected Phase 1 heading")
	}
	if !strings.Contains(prompt, "Phase 2") {
		t.Errorf("expected Phase 2 heading")
	}
	if !strings.Contains(prompt, "Phase 3") {
		t.Errorf("expected Phase 3 heading")
	}
	if !strings.Contains(prompt, "Phase 4") {
		t.Errorf("expected Phase 4 heading")
	}
}

func TestBuildConsolidationPrompt_emptyExtra(t *testing.T) {
	memoryRoot := "/fake/mem"
	transcriptDir := "/fake/transcripts"

	prompt := BuildConsolidationPrompt(memoryRoot, transcriptDir, "")

	if strings.Contains(prompt, "Additional context") {
		t.Errorf("expected no additional context section when extra is empty")
	}
}

func TestBuildConsolidationPrompt_dirExistsGuidance(t *testing.T) {
	prompt := BuildConsolidationPrompt("/m", "/t", "extra")

	if !strings.Contains(prompt, dirExistsGuidance) {
		t.Errorf("expected prompt to include directory-exists guidance")
	}
}

func TestBuildConsolidationPrompt_summary(t *testing.T) {
	prompt := BuildConsolidationPrompt("/m", "/t", "extra")

	// Should end with the summary instruction
	if !strings.Contains(prompt, "Return a brief summary") {
		t.Errorf("expected prompt to request a summary")
	}
}
