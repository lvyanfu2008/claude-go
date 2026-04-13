package paritytools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCronCreate_list_durable(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{ProjectRoot: dir, WorkDir: dir}
	cronMu.Lock()
	sessionBuf = nil
	cronMu.Unlock()
	in := `{"cron": "0 9 * * *", "prompt": "hi", "recurring": true, "durable": true}`
	out, isErr, err := CronCreateFromJSON([]byte(in), cfg)
	if err != nil || isErr {
		t.Fatalf("create: %v %v %s", err, isErr, out)
	}
	var cr map[string]any
	if err := json.Unmarshal([]byte(out), &cr); err != nil {
		t.Fatal(err)
	}
	d0 := cr["data"].(map[string]any)
	id, _ := d0["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	path := filepath.Join(dir, ".claude", "scheduled_tasks.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	listOut, isErr, err := CronListFromJSON([]byte("{}"), cfg)
	if err != nil || isErr {
		t.Fatalf("list: %v %v %s", err, isErr, listOut)
	}
	var li map[string]any
	if err := json.Unmarshal([]byte(listOut), &li); err != nil {
		t.Fatal(err)
	}
	d1 := li["data"].(map[string]any)
	jobs := d1["jobs"].([]any)
	if len(jobs) != 1 {
		t.Fatalf("jobs len %d", len(jobs))
	}
	del := `{"id": "` + id + `"}`
	_, _, err = CronDeleteFromJSON([]byte(del), cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sfile := string(b)
	if !strings.Contains(sfile, `"tasks"`) {
		t.Fatalf("expected tasks key: %s", sfile)
	}
	if !strings.Contains(sfile, `"tasks": []`) && !strings.Contains(sfile, `"tasks":[]`) {
		t.Fatalf("expected empty tasks after delete: %s", sfile)
	}
}
