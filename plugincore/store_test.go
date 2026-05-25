package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func makeTestTarball(manifestJSON string) io.Reader {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{Name: "plugin.json", Size: int64(len(manifestJSON)), Mode: 0644}
	tw.WriteHeader(hdr)
	tw.Write([]byte(manifestJSON))

	tw.Close()
	gw.Close()
	return &buf
}

func TestStore_Install(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	manifestJSON := `{"name": "test-plugin", "version": "1.0.0", "skills": ["skills/my-skill"]}`

	payload := io.NopCloser(makeTestTarball(manifestJSON))
	plugin := &Plugin{
		Meta:     PluginMeta{ID: "test-plugin@claude-plugins-official", Name: "test-plugin", Version: "1.0.0"},
		Manifest: PluginManifest{Name: "test-plugin", Version: "1.0.0", Skills: []string{"skills/my-skill"}},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/test-plugin"},
		Payload:  payload,
	}

	st := NewDiskStore()
	installed, err := st.Install(context.Background(), *plugin, cacheDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if installed.InstallPath == "" {
		t.Fatal("expected non-empty InstallPath")
	}

	manifestPath := filepath.Join(installed.InstallPath, "plugin.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("plugin.json not extracted")
	}

	// The marketplace name is derived from the first segment of Source.Repo ("claude-plugins-official/test-plugin" → "claude-plugins-official").
	marketplace := "claude-plugins-official"
	currentPath := filepath.Join(cacheDir, marketplace, "test-plugin", "current")
	target, err := os.Readlink(currentPath)
	if err != nil {
		t.Fatalf("current symlink not created: %v", err)
	}
	if target != installed.InstallPath {
		t.Errorf("current symlink points to %q, expected %q", target, installed.InstallPath)
	}
}

func TestStore_Install_CurrentVersionFallback(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	manifestJSON := `{"name": "fallback-plugin", "version": "2.0.0"}`

	st := NewDiskStore()
	payload := io.NopCloser(makeTestTarball(manifestJSON))
	plugin := &Plugin{
		Meta:     PluginMeta{ID: "fallback-plugin@claude-plugins-official", Name: "fallback-plugin", Version: "2.0.0"},
		Manifest: PluginManifest{Name: "fallback-plugin", Version: "2.0.0"},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/fallback-plugin"},
		Payload:  payload,
	}

	installed, err := st.Install(context.Background(), *plugin, cacheDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	marketplace := "claude-plugins-official"
	pluginPath := filepath.Join(cacheDir, marketplace, "fallback-plugin")

	// Remove the symlink to simulate Windows (no symlink support).
	os.Remove(filepath.Join(pluginPath, "current"))
	// current_version should be written as fallback on Windows.
	// On macOS/Linux the symlink succeeds, so we simulate the fallback.
	os.WriteFile(filepath.Join(pluginPath, "current_version"), []byte("2.0.0"), 0644)

	// ResolveCurrentPath should fall back to current_version.
	realPath, err := ResolveCurrentPath(pluginPath)
	if err != nil {
		t.Fatalf("ResolveCurrentPath: %v", err)
	}
	if realPath != installed.InstallPath {
		t.Errorf("ResolveCurrentPath = %q, want %q", realPath, installed.InstallPath)
	}

	// ListInstalled should also work with the fallback.
	list, err := st.ListInstalled(context.Background(), cacheDir)
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(list))
	}
	if list[0].Name != "fallback-plugin" {
		t.Errorf("expected 'fallback-plugin', got %q", list[0].Name)
	}
}

func TestStore_InstallAndList(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	manifestJSON := `{"name": "listable", "version": "1.0.0"}`

	st := NewDiskStore()
	payload := io.NopCloser(makeTestTarball(manifestJSON))
	plugin := &Plugin{
		Meta:     PluginMeta{ID: "listable@claude-plugins-official", Name: "listable", Version: "1.0.0"},
		Manifest: PluginManifest{Name: "listable", Version: "1.0.0"},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/listable"},
		Payload:  payload,
	}

	if _, err := st.Install(context.Background(), *plugin, cacheDir); err != nil {
		t.Fatalf("Install: %v", err)
	}

	installed, err := st.ListInstalled(context.Background(), cacheDir)
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed plugin, got %d", len(installed))
	}
	if installed[0].Name != "listable" {
		t.Errorf("expected 'listable', got %q", installed[0].Name)
	}
}

func TestStore_Uninstall(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	manifestJSON := `{"name": "removable", "version": "1.0.0"}`

	st := NewDiskStore()
	payload := io.NopCloser(makeTestTarball(manifestJSON))
	plugin := &Plugin{
		Meta:     PluginMeta{ID: "removable@claude-plugins-official", Name: "removable", Version: "1.0.0"},
		Manifest: PluginManifest{Name: "removable", Version: "1.0.0"},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/removable"},
		Payload:  payload,
	}

	if _, err := st.Install(context.Background(), *plugin, cacheDir); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := st.Uninstall(context.Background(), cacheDir, "removable@claude-plugins-official"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	installed, err := st.ListInstalled(context.Background(), cacheDir)
	if err != nil {
		t.Fatalf("ListInstalled after uninstall: %v", err)
	}
	if len(installed) != 0 {
		t.Errorf("expected 0 plugins after uninstall, got %d", len(installed))
	}
}

func TestStore_GC_KeepsLatest(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	st := NewDiskStore()

	p1 := &Plugin{
		Meta:     PluginMeta{ID: "gc-test@claude-plugins-official", Name: "gc-test", Version: "1.0.0"},
		Manifest: PluginManifest{Name: "gc-test", Version: "1.0.0"},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/gc-test"},
		Payload:  io.NopCloser(makeTestTarball(`{"name":"gc-test","version":"1.0.0"}`)),
	}
	st.Install(context.Background(), *p1, cacheDir)

	p2 := &Plugin{
		Meta:     PluginMeta{ID: "gc-test@claude-plugins-official", Name: "gc-test", Version: "2.0.0"},
		Manifest: PluginManifest{Name: "gc-test", Version: "2.0.0"},
		Source:   Source{Type: SourceGitHubRelease, Repo: "claude-plugins-official/gc-test"},
		Payload:  io.NopCloser(makeTestTarball(`{"name":"gc-test","version":"2.0.0"}`)),
	}
	st.Install(context.Background(), *p2, cacheDir)

	removed, err := st.GC(context.Background(), cacheDir, 1)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	installed, _ := st.ListInstalled(context.Background(), cacheDir)
	if len(installed) != 1 || installed[0].Version != "2.0.0" {
		t.Errorf("expected only v2.0.0 remaining, got version %q", installed[0].Version)
	}
}
