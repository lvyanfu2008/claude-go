package paritytools

import (
	"encoding/json"
	"os"
	"testing"
)

func TestPlanMode_enterExit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := Config{ProjectRoot: dir}
	out1, isErr, err := EnterPlanModeFromJSON([]byte("{}"), cfg)
	if err != nil || isErr {
		t.Fatalf("enter: %v %v %s", err, isErr, out1)
	}
	p := cfg.PlanModePath()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var st map[string]any
	if err := json.Unmarshal(b, &st); err != nil {
		t.Fatal(err)
	}
	if st["active"] != true {
		t.Fatalf("expected active: %v", st)
	}
	out2, isErr, err := ExitPlanModeFromJSON([]byte(`{"allowedPrompts":[{"tool":"Bash","prompt":"tests"}]}`), cfg)
	if err != nil || isErr {
		t.Fatalf("exit: %v %v %s", err, isErr, out2)
	}
	b2, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b2, &st); err != nil {
		t.Fatal(err)
	}
	if st["active"] != false {
		t.Fatalf("expected inactive: %v", st)
	}
}
