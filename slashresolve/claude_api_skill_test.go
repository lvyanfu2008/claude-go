package slashresolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectClaudeAPILanguage_goMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectClaudeAPILanguage(dir); got != "go" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildClaudeAPIPrompt_includesDoc(t *testing.T) {
	skill := "# Title\n\n## Reading Guide\n\nignore"
	files := map[string]string{
		"go/claude-api/README.md": "# API\nhello",
		"shared/models.md":        "# M",
	}
	out := BuildClaudeAPIPrompt("go", "hi", skill, files, ClaudeAPIModelVars)
	if !strings.Contains(out, "<doc path=\"go/claude-api/README.md\">") {
		t.Fatal(out[:500])
	}
	if !strings.Contains(out, "## User Request") {
		t.Fatal("missing user request")
	}
}
