package localtools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapGrepToolOutputToToolResultContent_contentMode(t *testing.T) {
	const raw = `{"mode":"content","numFiles":0,"filenames":[],"content":"conversation-runtime/query/a.go:17:\tfoo","numLines":1}`
	got, err := MapGrepToolOutputToToolResultContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := "conversation-runtime/query/a.go:17:\tfoo"
	if got != want {
		t.Fatalf("unexpected block content: %q want %q", got, want)
	}
}

func TestGrepFromJSON_contentRelativePath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "x.go")
	if err := os.WriteFile(file, []byte("hello GOU_QUERY_STREAMING_FORCE_ANTHROPIC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"pattern":"GOU_QUERY_STREAMING","path":` + mustJSONEscape(dir) + `,"output_mode":"content"}`)
	out, isErr, err := GrepFromJSON(context.Background(), raw, []string{dir})
	if err != nil || isErr {
		t.Fatalf("grep: err=%v isErr=%v", err, isErr)
	}
	var o struct {
		Mode    string `json:"mode"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(out), &o); err != nil {
		t.Fatal(err)
	}
	if o.Mode != "content" {
		t.Fatalf("mode %q", o.Mode)
	}
	if strings.Contains(o.Content, dir) {
		t.Fatalf("expected relative content, got %q", o.Content)
	}
	if !strings.Contains(o.Content, "pkg/x.go") && !strings.Contains(o.Content, `pkg/x.go`) {
		t.Fatalf("expected pkg/x.go in content, got %q", o.Content)
	}
	block, err := MapGrepToolOutputToToolResultContent(out)
	if err != nil {
		t.Fatal(err)
	}
	if block != o.Content {
		t.Fatalf("map mismatch: block=%q content=%q", block, o.Content)
	}
}

func mustJSONEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
