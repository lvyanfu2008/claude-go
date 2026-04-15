package skilltools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParityToolRunner_REPL_singleRead(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(p, []byte("xyzzy"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{
		"tool": "Read",
		"input": map[string]any{
			"file_path": p,
		},
	})
	r := &ParityToolRunner{DemoToolRunner: DemoToolRunner{}, WorkDir: dir, LocalBashDefault: true}
	out, isErr, err := r.Run(context.Background(), "REPL", "repl-1", raw)
	if err != nil {
		t.Fatal(err)
	}
	if isErr {
		t.Fatalf("isErr: %q", out)
	}
	if !strings.Contains(out, "[Read]") || !strings.Contains(out, "xyzzy") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestParityToolRunner_REPL_rejectsOuterSkill(t *testing.T) {
	dir := t.TempDir()
	raw, _ := json.Marshal(map[string]any{
		"tool":  "Skill",
		"input": map[string]any{"skill": "x"},
	})
	r := &ParityToolRunner{DemoToolRunner: DemoToolRunner{}, WorkDir: dir, LocalBashDefault: true}
	_, _, err := r.Run(context.Background(), "REPL", "repl-2", raw)
	if err == nil {
		t.Fatal("expected error for Skill inside REPL")
	}
}

func TestParityToolRunner_REPL_batch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.go")
	if err := os.WriteFile(p, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{
		"batch": []map[string]any{
			{"name": "Read", "input": map[string]any{"file_path": p}},
			{"name": "Read", "input": map[string]any{"file_path": p}},
		},
	})
	r := &ParityToolRunner{DemoToolRunner: DemoToolRunner{}, WorkDir: dir, LocalBashDefault: true}
	out, isErr, err := r.Run(context.Background(), "REPL", "repl-3", raw)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v out=%q", err, isErr, out)
	}
	if c := strings.Count(out, "[Read]"); c < 2 {
		t.Fatalf("want two Read blocks, got %q", out)
	}
}
