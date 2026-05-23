package plugins

import (
	"encoding/json"
	"testing"
)

func TestPluginManifestUnmarshal(t *testing.T) {
	data := []byte(`{
		"name": "superpowers",
		"version": "5.1.0",
		"skills": ["skills/brainstorming", "skills/debugging"],
		"hooks": {"SessionStart": [{"matcher": "startup", "hooks": [{"type": "command", "command": "./hooks/session-start"}]}]}
	}`)
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if m.Name != "superpowers" {
		t.Errorf("expected name 'superpowers', got %q", m.Name)
	}
	if m.Version != "5.1.0" {
		t.Errorf("expected version '5.1.0', got %q", m.Version)
	}
	if len(m.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(m.Skills))
	}
	if m.Hooks == nil {
		t.Error("expected non-nil hooks")
	}
}

func TestPluginManifestMinimal(t *testing.T) {
	data := []byte(`{"name": "minimal", "version": "1.0.0"}`)
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("minimal manifest should parse: %v", err)
	}
	if m.Name != "minimal" {
		t.Errorf("expected name 'minimal', got %q", m.Name)
	}
}

func TestEnginesConstraintUnmarshal(t *testing.T) {
	data := []byte(`{"name":"test","version":"1.0","engines":{"claude":">=2.0.0"}}`)
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Engines == nil || m.Engines.Claude != ">=2.0.0" {
		t.Errorf("expected engines.claude '>=2.0.0', got %+v", m.Engines)
	}
}

func TestSourceLocalPathSerialization(t *testing.T) {
	s := Source{Type: SourceLocalPath, Path: "/home/user/my-plugin"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 Source
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s2.Type != SourceLocalPath || s2.Path != "/home/user/my-plugin" {
		t.Errorf("round-trip failed: %+v", s2)
	}
}

func TestSourceSerialization(t *testing.T) {
	s := Source{Type: SourceGitHubRelease, Repo: "obra/superpowers"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 Source
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s2.Type != SourceGitHubRelease || s2.Repo != "obra/superpowers" {
		t.Errorf("round-trip failed: %+v", s2)
	}
}
