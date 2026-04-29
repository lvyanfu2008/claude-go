package extractmemories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractMemoriesLogToDiagLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")
	t.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", logPath)
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG", "")

	fileExtractMemoriesLogf("test line %d", 1)
	fileExtractMemoriesLogf("test line %d", 2)

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "test line 1") || !strings.Contains(s, "test line 2") {
		t.Fatalf("expected both lines, got:\n%s", s)
	}
	if !strings.Contains(s, "[extract-memories]") {
		t.Fatalf("expected tag, got:\n%s", s)
	}
}

func TestExtractMemoriesLogExplicitlyOff(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")
	t.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", logPath)
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG", "0")

	fileExtractMemoriesLogf("should not appear")

	b, err := os.ReadFile(logPath)
	if err != nil {
		// File not created at all — that's fine, means nothing was written
		return
	}
	if strings.Contains(string(b), "should not appear") {
		t.Fatalf("log line should not appear when logging is off")
	}
}

func TestExtractMemoriesLogExplicitlyOffClaudeCodeAlias(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")
	t.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", logPath)
	t.Setenv("CLAUDE_CODE_EXTRACT_MEMORIES_LOG", "false")

	fileExtractMemoriesLogf("should not appear")

	b, err := os.ReadFile(logPath)
	if err != nil {
		return
	}
	if strings.Contains(string(b), "should not appear") {
		t.Fatalf("log line should not appear when CLAUDE_CODE_EXTRACT_MEMORIES_LOG=false")
	}
}
