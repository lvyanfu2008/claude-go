package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_PluginJSON(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "plugin.json")
	data := `{"name": "test-plugin", "version": "2.0.0", "skills": ["skills/foo"]}`
	if err := os.WriteFile(manifestPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(tmp)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Name != "test-plugin" {
		t.Errorf("expected 'test-plugin', got %q", m.Name)
	}
	if m.Version != "2.0.0" {
		t.Errorf("expected '2.0.0', got %q", m.Version)
	}
}

func TestLoadManifest_PackageJSONFallback(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "package.json")
	data := `{"name": "legacy-plugin", "version": "1.0.0"}`
	if err := os.WriteFile(manifestPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(tmp)
	if err != nil {
		t.Fatalf("LoadManifest with package.json fallback: %v", err)
	}
	if m.Name != "legacy-plugin" {
		t.Errorf("expected 'legacy-plugin', got %q", m.Name)
	}
}

func TestLoadManifest_PluginJSONPreferred(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "plugin.json"), []byte(`{"name": "newer", "version": "2.0.0"}`), 0644)
	os.WriteFile(filepath.Join(tmp, "package.json"), []byte(`{"name": "older", "version": "1.0.0"}`), 0644)

	m, err := LoadManifest(tmp)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Name != "newer" {
		t.Errorf("plugin.json should take priority, got %q", m.Name)
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := LoadManifest(tmp)
	if err == nil {
		t.Fatal("expected error when no manifest found")
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "plugin.json"), []byte(`not json`), 0644)
	_, err := LoadManifest(tmp)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
