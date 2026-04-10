package settingsfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeGouDemoLocalePrefs_projectLayers(t *testing.T) {
	dir := t.TempDir()
	cl := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(cl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(cl, "settings.go.json"),
		[]byte(`{"language":"es"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(cl, "settings.local.json"),
		[]byte(`{"language":"fr","outputStyle":"Explanatory"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	lang, style, err := MergeGouDemoLocalePrefs(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if lang != "fr" {
		t.Fatalf("language: got %q want fr", lang)
	}
	if style != "Explanatory" {
		t.Fatalf("outputStyle: got %q want Explanatory", style)
	}
}

func TestMergeGouDemoLocalePrefs_includeUser_projectOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	ucl := filepath.Join(home, ".claude")
	if err := os.MkdirAll(ucl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(ucl, "settings.json"),
		[]byte(`{"language":"de"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	proj := t.TempDir()
	pcl := filepath.Join(proj, ".claude")
	if err := os.MkdirAll(pcl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(pcl, "settings.go.json"),
		[]byte(`{"language":"es"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	lang, _, err := MergeGouDemoLocalePrefs(proj, true)
	if err != nil {
		t.Fatal(err)
	}
	if lang != "es" {
		t.Fatalf("language: got %q want es (project go settings override user)", lang)
	}
}

func TestMergeGouDemoLocalePrefs_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	cl := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(cl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cl, "settings.go.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := MergeGouDemoLocalePrefs(dir, false)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
