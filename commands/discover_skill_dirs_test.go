package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiscoverSkillDirsForPaths_nestedSkills(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	pkg := filepath.Join(repo, "src", "pkg")
	skillDir := filepath.Join(pkg, ".claude", "skills", "myskill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pkg, "a.go")
	if err := os.WriteFile(file, []byte("package"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := DiscoverSkillDirsForPaths([]string{file}, repo, nil)
	if len(out) != 1 {
		t.Fatalf("want 1 dir, got %#v", out)
	}
	want := filepath.Join(pkg, ".claude", "skills")
	if filepath.Clean(out[0]) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", out[0], want)
	}
}

func TestDiscoverSkillDirsForPaths_skipsGitignored(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	nm := filepath.Join(repo, "node_modules", "p")
	skillDir := filepath.Join(nm, ".claude", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyTpl := filepath.Join(t.TempDir(), "empty-git-template")
	if err := os.MkdirAll(emptyTpl, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) error {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		return c.Run()
	}
	if err := run(repo, "git", "init", "--template="+emptyTpl); err != nil {
		t.Skip("git init:", err)
	}
	if err := run(repo, "git", "add", ".gitignore"); err != nil {
		t.Fatal(err)
	}
	if err := run(repo, "git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "init"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nm, "x.go")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := DiscoverSkillDirsForPaths([]string{file}, repo, nil)
	for _, d := range out {
		if filepath.Clean(d) == filepath.Clean(skillDir) {
			t.Fatalf("expected gitignored path skipped, got %q", d)
		}
	}
}

func TestDiscoverSkillDirsForPaths_persistentSeen(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	a := filepath.Join(repo, "a")
	skillDir := filepath.Join(a, ".claude", "skills", "s")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(a, "1.go")
	f2 := filepath.Join(a, "2.go")
	for _, f := range []string{f1, f2} {
		if err := os.WriteFile(f, []byte("p"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	seen := make(map[string]struct{})
	out1 := DiscoverSkillDirsForPaths([]string{f1}, repo, seen)
	out2 := DiscoverSkillDirsForPaths([]string{f2}, repo, seen)
	if len(out1) != 1 {
		t.Fatalf("out1=%#v", out1)
	}
	if len(out2) != 0 {
		t.Fatalf("second call should not rediscover same skill dir, got %#v", out2)
	}
}
