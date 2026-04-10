package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Go os.ReadDir returns names sorted lexically; same baseDir yields aaa before zzz.
func TestLoadSkillsFromSkillDirectories_withinDirLexicalOrder(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, ".claude", "skills")
	for _, name := range []string{"zzz", "aaa"} {
		d := filepath.Join(base, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := LoadSkillsFromSkillDirectories([]string{base})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2, got %d", len(out))
	}
	if out[0].Name != "aaa" || out[1].Name != "zzz" {
		t.Fatalf("want lexical [aaa zzz], got [%s %s]", out[0].Name, out[1].Name)
	}
}

// TS getDynamicSkills() uses Map iteration order: shallow skill dirs processed first in addSkillDirectories
// merge loop, so first-seen names are not sorted alphabetically.
func TestLoadSkillsFromSkillDirectories_orderMatchesTSMapNotNameSort(t *testing.T) {
	tmp := t.TempDir()
	shallow := filepath.Join(tmp, "proj", ".claude", "skills")
	deep := filepath.Join(tmp, "proj", "nested", ".claude", "skills")
	for _, d := range []string{
		filepath.Join(shallow, "zebra"),
		filepath.Join(deep, "apple"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(shallow, "zebra", "SKILL.md"), []byte("---\ndescription: z\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "apple", "SKILL.md"), []byte("---\ndescription: a\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Deepest first (same as DiscoverSkillDirsForPaths / TS discoverSkillDirsForPaths).
	dirs := []string{deep, shallow}
	out, err := LoadSkillsFromSkillDirectories(dirs)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 commands, got %d", len(out))
	}
	if out[0].Name != "zebra" || out[1].Name != "apple" {
		t.Fatalf("want merge order [zebra apple], got [%s %s] (alphabetical would be apple zebra)",
			out[0].Name, out[1].Name)
	}
}

func TestLoadSkillsFromSkillDirectories_deeperWinsSameName(t *testing.T) {
	tmp := t.TempDir()
	shallow := filepath.Join(tmp, "a", ".claude", "skills", "dup")
	deep := filepath.Join(tmp, "a", "b", ".claude", "skills", "dup")
	for _, d := range []string{shallow, deep} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(shallow, "SKILL.md"), []byte("---\ndescription: shallow\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "SKILL.md"), []byte("---\ndescription: deep\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// deepest first (DiscoverSkillDirsForPaths order)
	dirs := []string{
		filepath.Join(tmp, "a", "b", ".claude", "skills"),
		filepath.Join(tmp, "a", ".claude", "skills"),
	}
	out, err := LoadSkillsFromSkillDirectories(dirs)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 command, got %d", len(out))
	}
	if out[0].Description != "deep" {
		t.Fatalf("deeper skill should win, got %q", out[0].Description)
	}
}

func TestLoadDynamicSkillCommandsForPaths_respectsPluginLock(t *testing.T) {
	tmp := t.TempDir()
	f := false
	opts := LoadOptions{BareMode: &f, SkillsPluginOnlyLocked: true}
	out, err := LoadDynamicSkillCommandsForPaths([]string{filepath.Join(tmp, "x.go")}, tmp, opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected nil when plugin locked, got %d", len(out))
	}
}

func TestLoadAndGetCommandsWithFilePathsDynamic_merges(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(tmp, "repo")
	pkg := filepath.Join(repo, "src", "pkg")
	skillDir := filepath.Join(pkg, ".claude", "skills", "dyn")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: dyn\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pkg, "f.go")
	if err := os.WriteFile(file, []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	auth := DefaultConsoleAPIAuth()
	out, err := LoadAndGetCommandsWithFilePathsDynamic(context.Background(), repo, opts, auth, []string{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range out {
		if c.Name == "dyn" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected dynamic skill merged into command list")
	}
}
