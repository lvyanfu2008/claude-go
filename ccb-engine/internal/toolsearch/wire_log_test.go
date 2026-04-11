package toolsearch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogWireRound_writesWhenDiagOn(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "wire.txt")
	t.Setenv("CLAUDE_CODE_GO_TOOL_SEARCH_DIAG", "1")
	t.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", logFile)
	t.Setenv("ENABLE_TOOL_SEARCH", "")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")

	full := sampleToolsAPIStyle()
	cfg := BuildWireConfig("claude-sonnet-4-20250514", full, false, false)
	wired := ApplyWire(full, nil, cfg)
	LogWireRound(0, "claude-sonnet-4-20250514", nil, cfg, false, full, wired)

	b, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty diag log")
	}
	s := string(b)
	if !strings.Contains(s, "[ccb-engine toolsearch-wire]") {
		t.Fatalf("log: %q", s)
	}
	if !strings.Contains(s, "wired_names=") {
		t.Fatalf("log: %q", s)
	}
	if !strings.Contains(s, "wire_reason=") {
		t.Fatalf("log: %q", s)
	}
}
