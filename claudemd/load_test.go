package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildClaudeMdString_projectClaudeMd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Hi\nSee @./inc.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inc.md"), []byte("included body"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CODE_DISABLE_USER_MEMORY", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	out := BuildClaudeMdString(LoadOptions{OriginalCwd: dir})
	if !strings.Contains(out, "included body") {
		t.Fatalf("missing include expansion: %q", out)
	}
	if !strings.Contains(out, MemoryInstructionPrompt[:40]) {
		t.Fatal("missing instruction prefix")
	}
}

func TestParseFrontmatterPaths_globs(t *testing.T) {
	raw := "---\npaths: src/*.ts, docs/**\n---\nbody"
	_, globs := ParseFrontmatterPaths(raw)
	if len(globs) < 1 {
		t.Fatalf("globs %#v", globs)
	}
}

func TestBuildClaudeMdString_claudeMdExcludesOverride(t *testing.T) {
	dir := t.TempDir()
	claude := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	pat := filepath.ToSlash(claude)
	override := []string{pat}
	t.Setenv("CLAUDE_CODE_DISABLE_USER_MEMORY", "1")
	out := BuildClaudeMdString(LoadOptions{
		OriginalCwd:              dir,
		ClaudeMdExcludesOverride: &override,
	})
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty when CLAUDE.md excluded, got %q", out)
	}
}
