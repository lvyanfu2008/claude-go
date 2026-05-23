package plugins_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	plugins "goc/plugincore"
)

func TestEndToEnd_LocalPlugin(t *testing.T) {
	tmp := t.TempDir()

	// Create a minimal plugin on disk
	pluginDir := filepath.Join(tmp, "my-plugin")
	os.MkdirAll(filepath.Join(pluginDir, "skills", "my-skill"), 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"),
		[]byte(`{"name":"my-plugin","version":"0.1.0","skills":["skills/my-skill"]}`), 0644)
	os.WriteFile(filepath.Join(pluginDir, "skills", "my-skill", "SKILL.md"),
		[]byte("---\nname: my-skill\ndescription: A test skill\n---\n\n# My Skill\n\nTest content."), 0644)

	// Fetch via local source
	src := plugins.Source{Type: plugins.SourceLocalPath, Path: pluginDir}
	fetcher := plugins.NewLocalSource()
	plugin, err := fetcher.Fetch(context.Background(), src, "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer plugin.Payload.Close()

	if plugin.Manifest.Name != "my-plugin" {
		t.Errorf("expected 'my-plugin', got %q", plugin.Manifest.Name)
	}

	// Install
	cacheDir := filepath.Join(tmp, "cache")
	store := plugins.NewDiskStore()
	installed, err := store.Install(context.Background(), *plugin, cacheDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if installed.Manifest.Version != "0.1.0" {
		t.Errorf("expected 0.1.0, got %q", installed.Manifest.Version)
	}

	// Verify installed files exist
	files, _ := os.ReadDir(installed.InstallPath)
	if len(files) == 0 {
		t.Fatal("no files in installed plugin dir")
	}

	// List installed
	plugins, err := store.ListInstalled(context.Background(), cacheDir)
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Uninstall
	if err := store.Uninstall(context.Background(), cacheDir, plugins[0].ID); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	remaining, _ := store.ListInstalled(context.Background(), cacheDir)
	if len(remaining) != 0 {
		t.Errorf("expected 0 plugins after uninstall, got %d", len(remaining))
	}
}
