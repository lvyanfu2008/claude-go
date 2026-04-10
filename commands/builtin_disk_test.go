package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltinPrefixDirs_skillMd(t *testing.T) {
	base := t.TempDir()
	skillDir := filepath.Join(base, "builtin_extra", "overlay_skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := `---
description: from builtin overlay
---
# Overlay
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{}
	out := loadBuiltinPrefixDirs(base, seen)
	if len(out) != 1 {
		t.Fatalf("got %d commands", len(out))
	}
	if out[0].Name != "overlay_skill" {
		t.Fatalf("name %q", out[0].Name)
	}
	if out[0].Source == nil || *out[0].Source != "builtin" {
		t.Fatalf("source %#v", out[0].Source)
	}
}

func TestLoadBuiltinPrefixDirs_jsonArray(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "builtin_json_cmds")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := `[{"type":"prompt","name":"from_json_builtin","description":"x"}]`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{}
	out := loadBuiltinPrefixDirs(base, seen)
	if len(out) != 1 || out[0].Name != "from_json_builtin" {
		t.Fatalf("got %+v", out)
	}
}

func TestLoadBuiltinPrefixDirs_embedWinsDuplicate(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "builtin_dup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := `[{"type":"prompt","name":"help","description":"shadow"}]`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{"help": {}}
	out := loadBuiltinPrefixDirs(base, seen)
	if len(out) != 0 {
		t.Fatalf("expected skip duplicate, got %+v", out)
	}
}

func TestLoadBuiltinPrefixDirs_ignoresNonBuiltinPrefixDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "data_extra", "x.json")
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte(`[{"type":"prompt","name":"nope","description":"x"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := loadBuiltinPrefixDirs(base, map[string]struct{}{})
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}
