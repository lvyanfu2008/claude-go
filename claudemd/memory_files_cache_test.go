package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMemoryFiles_memoizeUntilClear(t *testing.T) {
	ClearMemoryFileCaches()
	ResetMemoryFilesCache("session_start")
	dir := t.TempDir()
	md := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(md, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := LoadOptions{OriginalCwd: dir}
	a1 := LoadMemoryFiles(opts)
	if len(a1) == 0 {
		t.Fatal("expected at least one project memory file")
	}
	if !strings.Contains(joinContent(a1), "alpha") {
		t.Fatalf("first load: %v", a1)
	}
	_ = os.WriteFile(md, []byte("beta"), 0o644)
	a2 := LoadMemoryFiles(opts)
	if !strings.Contains(joinContent(a2), "alpha") {
		t.Fatalf("memo should keep alpha, got %#v", a2)
	}
	ClearMemoryFileCaches()
	a3 := LoadMemoryFiles(opts)
	if !strings.Contains(joinContent(a3), "beta") {
		t.Fatalf("after clearMemoryFileCaches expected fresh beta, got %#v", a3)
	}
}

func joinContent(files []MemoryFileInfo) string {
	var b strings.Builder
	for _, f := range files {
		b.WriteString(f.Content)
	}
	return b.String()
}
