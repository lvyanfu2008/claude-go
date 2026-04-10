package submitfill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyIfEmpty_noopWhenSystemSet(t *testing.T) {
	t.Setenv("CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY", "1")
	msgs := json.RawMessage(`[{"role":"user","content":"hi"}]`)
	sys, out, err := ApplyIfEmpty("already", msgs, Options{FetchIfEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	if sys != "already" || string(out) != string(msgs) {
		t.Fatalf("got sys=%q msgs=%s", sys, out)
	}
}

func TestApplyIfEmpty_noopWhenFetchDisabled(t *testing.T) {
	t.Setenv("CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY", "")
	msgs := json.RawMessage(`[{"role":"user","content":"hi"}]`)
	sys, out, err := ApplyIfEmpty("", msgs, Options{FetchIfEmpty: false})
	if err != nil {
		t.Fatal(err)
	}
	if sys != "" || string(out) != string(msgs) {
		t.Fatalf("got sys=%q msgs=%s", sys, out)
	}
}

func TestApplyIfEmpty_buildsSystem(t *testing.T) {
	t.Setenv("CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY", "")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-01-01")
	dir := t.TempDir()
	msgs := json.RawMessage(`[{"role":"user","content":"hello"}]`)
	sys, out, err := ApplyIfEmpty("", msgs, Options{
		FetchIfEmpty: true,
		Cwd:          dir,
		ModelID:      "test-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sys, "interactive agent") {
		t.Fatalf("expected built system, got %q", preview(sys, 200))
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) < 1 {
		t.Fatalf("messages: %#v", arr)
	}
	// currentDate in user context usually produces a prepended system-reminder user message.
}

func TestApplyIfEmpty_prependReminderWithClaudeMd(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-01-02")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# P\nhello from md"), 0o600); err != nil {
		t.Fatal(err)
	}
	msgs := json.RawMessage(`[{"role":"user","content":"x"}]`)
	sys, out, err := ApplyIfEmpty("", msgs, Options{FetchIfEmpty: true, Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if sys == "" {
		t.Fatal("empty system")
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) < 2 {
		t.Fatalf("expected prepended user message, got %#v", arr)
	}
}

func preview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
