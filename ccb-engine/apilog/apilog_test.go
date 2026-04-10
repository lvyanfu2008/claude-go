package apilog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/ccb-engine/debugpath"
)

func TestLogRequestBody_skipsWhenUnset(t *testing.T) {
	_ = os.Unsetenv("CLAUDE_CODE_LOG_API_REQUEST_BODY")
	_ = os.Unsetenv("CLAUDE_CODE_DEBUG_LOG_FILE")
	// Should not panic; no file required
	LogRequestBody("test", []byte(`{}`))
}

func TestLogRequestBody_writesWhenSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CODE_LOG_API_REQUEST_BODY", "1")
	t.Setenv("CLAUDE_CODE_DEBUG_LOGS_DIR", dir)
	path := filepath.Join(dir, debugpath.SessionID()+".txt")
	LogRequestBody("POST /x", []byte(`{"a":1}`))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if len(b) < 20 || !strings.Contains(s, "API_REQUEST_BODY") || !strings.Contains(s, `"a"`) {
		t.Fatalf("unexpected file: %s", s)
	}
}

func TestLogRequestBody_defaultPathUnderHomeClaudeDebug(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_LOG_API_REQUEST_BODY", "1")
	t.Setenv("CLAUDE_CODE_DEBUG_LOG_FILE", "")
	t.Setenv("CLAUDE_CODE_DEBUG_LOGS_DIR", "")
	path := filepath.Join(home, ".claude", "debug", debugpath.SessionID()+".txt")
	LogRequestBody("POST /default", []byte(`{"path":"default"}`))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "API_REQUEST_BODY") || !strings.Contains(s, `"path"`) {
		t.Fatalf("unexpected file: %s", s)
	}
}

func TestResolvedLogPath_default(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_DEBUG_LOG_FILE", "")
	t.Setenv("CLAUDE_CODE_DEBUG_LOGS_DIR", "")
	want := filepath.Join(home, ".claude", "debug", debugpath.SessionID()+".txt")
	if got := ResolvedLogPath(); got != want {
		t.Fatalf("ResolvedLogPath: got %q want %q", got, want)
	}
}

func TestPrepareIfEnabled_createsClaudeDebugDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_LOG_API_REQUEST_BODY", "1")
	t.Setenv("CLAUDE_CODE_DEBUG_LOG_FILE", "")
	t.Setenv("CLAUDE_CODE_DEBUG_LOGS_DIR", "")
	PrepareIfEnabled()
	dbg := filepath.Join(home, ".claude", "debug")
	st, err := os.Stat(dbg)
	if err != nil || !st.IsDir() {
		t.Fatalf("expected directory %s: %v", dbg, err)
	}
	path := filepath.Join(dbg, debugpath.SessionID()+".txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected log file created: %v", err)
	}
}
