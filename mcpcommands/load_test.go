package mcpcommands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPath_empty(t *testing.T) {
	out, err := LoadFromPath("")
	if err != nil || out != nil {
		t.Fatalf("got (%v, %v), want (nil, nil)", out, err)
	}
}

func TestLoadFromPath_sample(t *testing.T) {
	p := filepath.Join("testdata", "sample_mcp_commands.json")
	out, err := LoadFromPath(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Type != "prompt" || out[0].Name != "mcp__demo__hello" {
		t.Fatalf("%+v", out[0])
	}
	if out[0].LoadedFrom == nil || *out[0].LoadedFrom != "mcp" {
		t.Fatalf("loadedFrom %+v", out[0].LoadedFrom)
	}
}

func TestLoadFromEnv_missing(t *testing.T) {
	t.Setenv(EnvCommandsJSONPath, "")
	out, err := LoadFromEnv()
	if err != nil || len(out) != 0 {
		t.Fatalf("got %v err=%v", out, err)
	}
}

func TestLoadFromPath_repoStylePathWhenCwdIsGocModule(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "commands", "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(tmp, "commands", "data", "mcp_commands.json")
	if err := os.WriteFile(f, []byte(`[]`), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	out, err := LoadFromPath("goc/commands/data/mcp_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("got %v", out)
	}
}

func TestLoadFromEnv_set(t *testing.T) {
	p := filepath.Join("testdata", "sample_mcp_commands.json")
	t.Setenv(EnvCommandsJSONPath, p)
	out, err := LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "mcp__demo__hello" {
		t.Fatalf("%+v", out)
	}
}
