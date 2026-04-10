package paritytools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTaskOutput_notReady(t *testing.T) {
	t.Parallel()
	cfg := Config{ProjectRoot: t.TempDir(), SessionID: "s2"}
	in := `{"task_id": "t1", "block": false}`
	out, isErr, err := TaskOutputFromJSON(context.Background(), []byte(in), cfg)
	if err != nil || isErr {
		t.Fatalf("not_ready: %v %v %s", err, isErr, out)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["retrieval_status"] != "not_ready" {
		t.Fatalf("got %v", v)
	}
}

func TestTaskOutput_readsFile(t *testing.T) {
	t.Parallel()
	cfg := Config{ProjectRoot: t.TempDir(), SessionID: "s4"}
	dir := cfg.TasksDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "abc.output")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := `{"task_id": "abc", "block": false}`
	out, isErr, err := TaskOutputFromJSON(context.Background(), []byte(in), cfg)
	if err != nil || isErr {
		t.Fatalf("read: %v %v %s", err, isErr, out)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["retrieval_status"] != "success" {
		t.Fatalf("got %v", v)
	}
}

func TestTaskStop_writesStopFile(t *testing.T) {
	t.Parallel()
	cfg := Config{ProjectRoot: t.TempDir(), SessionID: "s5"}
	in := `{"task_id": "x1"}`
	_, _, err := TaskStopFromJSON([]byte(in), cfg)
	if err != nil {
		t.Fatal(err)
	}
	stop := filepath.Join(cfg.TasksDir(), "x1.stop")
	if _, err := os.Stat(stop); err != nil {
		t.Fatal(err)
	}
}
