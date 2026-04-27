package extractmemories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractMemoriesLogFileAppend(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nested", "extract-mem.log")
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG_FILE", logPath)
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG", "") // not off

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
	logPath := filepath.Join(dir, "off.log")
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG_FILE", logPath)
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG", "0")

	fileExtractMemoriesLogf("should not appear")

	if _, err := os.Stat(logPath); err == nil {
		b, _ := os.ReadFile(logPath)
		t.Fatalf("log file should not be created, got: %q", string(b))
	}
}

func TestExtractMemoriesLogFileClaudeCodeAlias(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "a.log")
	t.Setenv("GOC_EXTRACT_MEMORIES_LOG_FILE", "")
	t.Setenv("CLAUDE_CODE_EXTRACT_MEMORIES_LOG_FILE", logPath)

	fileExtractMemoriesLogf("alias")

	b, err := os.ReadFile(logPath)
	if err != nil || !strings.Contains(string(b), "alias") {
		t.Fatalf("read log: %v body=%q", err, string(b))
	}
}
