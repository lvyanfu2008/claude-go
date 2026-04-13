package paritytools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTodoWrite_roundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := Config{ProjectRoot: dir, SessionID: "s1"}
	in := `{"todos":[{"content":"a","status":"pending","activeForm":"doing a"}]}`
	out, isErr, err := TodoWriteFromJSON([]byte(in), cfg)
	if err != nil || isErr {
		t.Fatalf("write: %v %v %s", err, isErr, out)
	}
	path := cfg.TodoFilePath()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	in2 := `{"todos":[{"content":"b","status":"completed","activeForm":"done b"}]}`
	_, _, err = TodoWriteFromJSON([]byte(in2), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data wrapper, got %#v", payload)
	}
	old, _ := data["oldTodos"].([]any)
	if old != nil && len(old) != 0 {
		t.Fatalf("expected empty oldTodos, got %d", len(old))
	}
	if filepath.Dir(path) != filepath.Join(dir, ".claude") {
		t.Fatalf("bad dir %s", filepath.Dir(path))
	}
}
