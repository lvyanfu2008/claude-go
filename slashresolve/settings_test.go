package slashresolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeEnabledPlugins_order(t *testing.T) {
	root := t.TempDir()
	cl := filepath.Join(root, ".claude")
	if err := os.MkdirAll(cl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cl, "settings.json"), []byte(`{"enabledPlugins":{"a":true,"b":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cl, "settings.local.json"), []byte(`{"enabledPlugins":{"b":true,"c":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgHome := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfgHome)
	if err := os.WriteFile(filepath.Join(cfgHome, "settings.json"), []byte(`{"enabledPlugins":{"base":true,"a":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := MergeEnabledPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	if m["base"] != true || m["a"] != true || m["b"] != true || m["c"] != true {
		t.Fatalf("merged: %#v", m)
	}
}
