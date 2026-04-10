package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStopBoundary_NestedRepoUsesSessionRoot(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "parent")
	nested := filepath.Join(parent, "nested")
	if err := os.MkdirAll(filepath.Join(nested, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(nested, "sub")
	got := resolveStopBoundary(cwd, parent)
	if normalizePathForComparison(got) != normalizePathForComparison(parent) {
		t.Fatalf("stop boundary: got %q want %q", got, parent)
	}
}

func TestResolveStopBoundary_SameRepoUsesNearestGit(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(tmp, "a", "b")
	got := resolveStopBoundary(cwd, cwd)
	if normalizePathForComparison(got) != normalizePathForComparison(tmp) {
		t.Fatalf("stop boundary: got %q want %q", got, tmp)
	}
}

func TestProjectClaudeSubdirs_StopsAtResolveBoundary(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "parent")
	nested := filepath.Join(parent, "nested")
	if err := os.MkdirAll(filepath.Join(nested, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Only parent has .claude/skills — must be visible when session is parent (walk continues past nested .git)
	parentSkills := filepath.Join(parent, ".claude", "skills")
	if err := os.MkdirAll(parentSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(nested, "work")
	dirs, err := projectClaudeSubdirs(cwd, "skills", parent)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, d := range dirs {
		if normalizePathForComparison(d) == normalizePathForComparison(parentSkills) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parent skills dir in %v", dirs)
	}
}
