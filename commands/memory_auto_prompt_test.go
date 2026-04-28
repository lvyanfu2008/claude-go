package commands

import (
	"strings"
	"testing"

	"goc/memdir"
)

func TestBuildAutoMemoryPromptKairosRequiresKairosActive(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "0")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("FEATURE_TEAMMEM", "0")

	if !memdir.IsAutoMemoryEnabled() {
		t.Fatal("test expects auto memory enabled")
	}

	dir := t.TempDir()
	base := GouDemoSystemOpts{Cwd: dir, MemorySkipIndex: false}

	withKairos := base
	withKairos.KairosActive = true
	outOn := BuildAutoMemoryPrompt(withKairos)
	if !strings.Contains(outOn, "long-lived") {
		t.Fatalf("expected KAIROS daily template when KairosActive=true, got prefix %q", truncate(outOn, 120))
	}

	noKairos := base
	noKairos.KairosActive = false
	outOff := BuildAutoMemoryPrompt(noKairos)
	if strings.Contains(outOff, "long-lived") {
		t.Fatal("expected non-KAIROS auto memory template when KairosActive=false")
	}
	if !strings.Contains(outOff, "Types of memory") {
		t.Fatalf("expected default auto memory template when KairosActive=false, got prefix %q", truncate(outOff, 120))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
