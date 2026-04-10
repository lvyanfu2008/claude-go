package settingsfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindClaudeProjectRoot_walksUpFromNestedDir(t *testing.T) {
	repo := t.TempDir()
	sub := filepath.Join(repo, "goc", "cmd", "nested")
	if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".claude", "settings.go.json"), []byte(`{"env":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindClaudeProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != repoAbs {
		t.Fatalf("want repo root %q, got %q", repoAbs, got)
	}
}

func TestFindClaudeProjectRoot_tsSettingsJsonDoesNotAnchor(t *testing.T) {
	repo := t.TempDir()
	sub := filepath.Join(repo, "pkg", "nested")
	if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	// TS-only project file: must not define Go project root for [FindClaudeProjectRoot].
	if err := os.WriteFile(filepath.Join(repo, ".claude", "settings.json"), []byte(`{"env":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindClaudeProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("TS-only settings.json must not anchor Go root: want %q, got %q", want, got)
	}
}

func TestFindClaudeProjectRootAny_findsViaTSsettingsJson(t *testing.T) {
	repo := t.TempDir()
	sub := filepath.Join(repo, "goc", "cmd", "nested")
	if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".claude", "settings.json"), []byte(`{"env":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindClaudeProjectRootAny(sub)
	if err != nil {
		t.Fatal(err)
	}
	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != repoAbs {
		t.Fatalf("FindClaudeProjectRootAny want repo root %q, got %q", repoAbs, got)
	}
}

func TestFindClaudeProjectRoot_noMarkerUsesStart(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "empty", "nest")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindClaudeProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
