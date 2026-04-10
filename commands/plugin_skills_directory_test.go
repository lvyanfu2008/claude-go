package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPluginSkillMD = `---
name: x
description: "d"
---
body
`

func TestLoadSkillsFromDirectory_directSKILLmd(t *testing.T) {
	root := t.TempDir()
	skillsPath := filepath.Join(root, "my-skill-pack")
	if err := os.MkdirAll(skillsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsPath, "SKILL.md"), []byte(testPluginSkillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded := map[string]struct{}{}
	manifest := json.RawMessage(`{"name":"p"}`)
	out, err := LoadSkillsFromDirectory(context.Background(), skillsPath, "p1", "p1@market", root, manifest, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Name != "p1:my-skill-pack" {
		t.Fatalf("name %q", out[0].Name)
	}
	if out[0].Source == nil || *out[0].Source != "plugin" {
		t.Fatalf("source %+v", out[0].Source)
	}
	if out[0].ProgressMessage == nil || *out[0].ProgressMessage != "loading" {
		t.Fatalf("progress %+v", out[0].ProgressMessage)
	}
}

func TestLoadSkillsFromDirectory_subdirs(t *testing.T) {
	root := t.TempDir()
	skillsPath := filepath.Join(root, "skills")
	for _, name := range []string{"a", "b"} {
		d := filepath.Join(skillsPath, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(testPluginSkillMD), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loaded := map[string]struct{}{}
	out, err := LoadSkillsFromDirectory(context.Background(), skillsPath, "plug", "plug@x", root, nil, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	names := map[string]bool{}
	for _, c := range out {
		names[c.Name] = true
	}
	if !names["plug:a"] || !names["plug:b"] {
		t.Fatalf("%v", names)
	}
}

func TestLoadSkillsFromDirectory_duplicateSkipped(t *testing.T) {
	root := t.TempDir()
	skillsPath := filepath.Join(root, "skills", "one")
	if err := os.MkdirAll(skillsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(skillsPath, "SKILL.md")
	if err := os.WriteFile(md, []byte(testPluginSkillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded := map[string]struct{}{}
	_, err := LoadSkillsFromDirectory(context.Background(), skillsPath, "p", "p@m", root, nil, loaded)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := LoadSkillsFromDirectory(context.Background(), skillsPath, "p", "p@m", root, nil, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(out2) != 0 {
		t.Fatalf("expected duplicate skip, got %d", len(out2))
	}
}

func TestLoadSkillsFromDirectory_substitutePluginRoot(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "plugin")
	skillsPath := filepath.Join(pluginRoot, "skills")
	if err := os.MkdirAll(filepath.Join(skillsPath, "s1"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
description: "x"
allowed-tools: "Bash(${CLAUDE_PLUGIN_ROOT}/bin/foo:*)"
---
ok
`
	if err := os.WriteFile(filepath.Join(skillsPath, "s1", "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := LoadSkillsFromDirectory(context.Background(), skillsPath, "p", "p@m", pluginRoot, nil, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || len(out[0].AllowedTools) != 1 {
		t.Fatal(out)
	}
	wantSub := filepath.ToSlash(filepath.Clean(pluginRoot)) + "/bin/foo"
	if !strings.Contains(out[0].AllowedTools[0], wantSub) {
		t.Fatalf("got %q want sub %q", out[0].AllowedTools[0], wantSub)
	}
}
