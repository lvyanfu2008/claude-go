package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindMarkdownFollowsSymlinkDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ; run on unix")
	}
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real", "nested")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(real, "note.md")
	if err := os.WriteFile(md, []byte("#"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(tmp, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tree, "via")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	out, err := findMarkdownFilesNative(tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 md via symlink walk, got %#v", out)
	}
}

func TestLoadSkillsFromDir_symlinkSkillDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ; run on unix")
	}
	tmp := t.TempDir()
	realSkill := filepath.Join(tmp, "actual", "myskill")
	if err := os.MkdirAll(realSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realSkill, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillsDir := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realSkill, filepath.Join(skillsDir, "myskill")); err != nil {
		t.Skip(err)
	}
	entries, err := loadSkillsFromDir(skillsDir, "projectSettings")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Cmd.Name != "myskill" {
		t.Fatalf("got %#v", entries)
	}
}
