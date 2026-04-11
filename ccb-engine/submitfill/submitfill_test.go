package submitfill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func userWirePlainText(t *testing.T, content any) string {
	t.Helper()
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("content: %#v", content)
	}
	var parts []string
	for _, b := range blocks {
		if typ, _ := b["type"].(string); typ == "text" {
			parts = append(parts, stringifyAny(b["text"]))
		}
	}
	return strings.Join(parts, "\n")
}

func stringifyAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		b, _ := json.Marshal(x)
		var s string
		_ = json.Unmarshal(b, &s)
		return s
	}
}

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
	if len(arr) != 1 {
		t.Fatalf("expected single user after TS-style merge of context + transcript, got %#v", arr)
	}
	content := userWirePlainText(t, arr[0]["content"])
	if !strings.Contains(content, "hello from md") || !strings.Contains(content, "\nx") {
		t.Fatalf("merged user should contain claudeMd and original line; got %q", preview(content, 400))
	}
}

func preview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
