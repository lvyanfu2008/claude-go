package localtools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadFromJSON_EnrichesENOENTWithSuggestion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "renderer.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	in := map[string]any{
		"file_path": "render.go",
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	_, _, err = ReadFromJSON(raw, []string{root}, nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Did you mean") {
		t.Fatalf("expected suggestion in error, got: %s", msg)
	}
	if !strings.Contains(msg, "renderer.go") {
		t.Fatalf("expected suggested file in error, got: %s", msg)
	}
}

func TestReadFromJSON_DevicePathBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix device path assertion")
	}
	in := map[string]any{
		"file_path": "/dev/null",
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	_, _, err = ReadFromJSON(raw, []string{"/"}, nil, nil)
	if err == nil {
		t.Fatalf("expected device path error")
	}
	if !strings.Contains(err.Error(), "device path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

