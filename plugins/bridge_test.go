package plugins

import (
	"os"
	"path/filepath"
	"testing"

	plugins "goc/plugincore"
)

func TestLoadSkillsFromPluginDir(t *testing.T) {
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(tmp, "skills", "brainstorming"), 0755)
	os.WriteFile(filepath.Join(tmp, "plugin.json"),
		[]byte(`{"name":"test","version":"1.0","skills":["skills/brainstorming"]}`), 0644)
	os.WriteFile(filepath.Join(tmp, "skills", "brainstorming", "SKILL.md"),
		[]byte("---\nname: test:brainstorming\ndescription: Test skill\n---\n\n# Test\n\nContent."), 0644)

	manifest, err := plugins.LoadManifest(tmp)
	if err != nil {
		t.Fatal(err)
	}

	cmds, err := LoadSkillsFromPlugin(tmp, manifest, "test", "test@local")
	if err != nil {
		t.Fatalf("LoadSkillsFromPlugin: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 skill command, got %d", len(cmds))
	}
	if cmds[0].Name != "test:brainstorming" {
		t.Errorf("expected 'test:brainstorming', got %q", cmds[0].Name)
	}
}

func TestLoadSkillsFromPluginDir_NoManifest(t *testing.T) {
	tmp := t.TempDir()
	_, err := LoadSkillsFromPluginDir(tmp, "test", "test@local", tmp)
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func TestLoadSkillsFromPlugin_NoSkills(t *testing.T) {
	tmp := t.TempDir()
	manifest := &plugins.PluginManifest{
		Name:    "test",
		Version: "1.0",
		Skills:  nil,
	}
	cmds, err := LoadSkillsFromPlugin(tmp, manifest, "test", "test@local")
	if err != nil {
		t.Fatalf("LoadSkillsFromPlugin: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 skill commands, got %d", len(cmds))
	}
}

func TestLoadSkillsFromPlugin_AutoDiscover(t *testing.T) {
	tmp := t.TempDir()

	// Simulate superpowers-like layout: skills/ subdir with skill dirs, no skills in manifest
	os.MkdirAll(filepath.Join(tmp, "skills", "brainstorming"), 0755)
	os.MkdirAll(filepath.Join(tmp, "skills", "debugging"), 0755)
	os.WriteFile(filepath.Join(tmp, "skills", "brainstorming", "SKILL.md"),
		[]byte("---\nname: test:brainstorming\ndescription: Brainstorming skill\n---\n\n# Brainstorming\n\nContent."), 0644)
	os.WriteFile(filepath.Join(tmp, "skills", "debugging", "SKILL.md"),
		[]byte("---\nname: test:debugging\ndescription: Debugging skill\n---\n\n# Debugging\n\nContent."), 0644)

	manifest := &plugins.PluginManifest{
		Name:    "test",
		Version: "1.0",
		Skills:  nil, // empty → auto-discover should kick in
	}

	cmds, err := LoadSkillsFromPlugin(tmp, manifest, "test", "test@local")
	if err != nil {
		t.Fatalf("LoadSkillsFromPlugin: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 auto-discovered skill commands, got %d", len(cmds))
	}
}
