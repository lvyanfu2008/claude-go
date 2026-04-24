package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAgentWorkDirFromCwd(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	_, err := resolveAgentWorkDirFromCwd(d)
	if err != nil {
		t.Fatalf("abs temp dir: %v", err)
	}
	_, err = resolveAgentWorkDirFromCwd("relative/path")
	if err == nil {
		t.Fatal("expected error for non-absolute path")
	}
	_, err = resolveAgentWorkDirFromCwd(filepath.Join(d, "nope"))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	f, err := os.CreateTemp(d, "file-")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	_, err = resolveAgentWorkDirFromCwd(f.Name())
	if err == nil {
		t.Fatal("expected error when cwd is a file, not a directory")
	}
}
