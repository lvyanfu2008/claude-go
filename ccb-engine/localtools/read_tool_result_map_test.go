package localtools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMapReadToolResultToAssistantText_mitigationAndCompact(t *testing.T) {
	t.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", "")
	raw, err := json.Marshal(ReadTextOutput{
		Type: "text",
		File: struct {
			FilePath   string `json:"filePath"`
			Content    string `json:"content"`
			NumLines   int    `json:"numLines"`
			StartLine  int    `json:"startLine"`
			TotalLines int    `json:"totalLines"`
		}{FilePath: "a.go", Content: "package main", NumLines: 1, StartLine: 1, TotalLines: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := MapReadToolResultToAssistantText(string(raw), &ReadToolResultMapOpts{MainLoopModel: "claude-sonnet-4-20250514"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Whenever you read a file") {
		t.Fatalf("expected cyber mitigation, got %q", out)
	}
	if !strings.HasPrefix(out, "1\tpackage main") {
		t.Fatalf("expected compact tab line prefix, got %q", out)
	}
}

func TestMapReadToolResultToAssistantText_opus46Exempt(t *testing.T) {
	raw := `{"type":"text","file":{"filePath":"x","content":"hi","numLines":1,"startLine":1,"totalLines":1}}`
	out, err := MapReadToolResultToAssistantText(raw, &ReadToolResultMapOpts{MainLoopModel: "claude-opus-4-6-20251101"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Whenever you read a file") {
		t.Fatalf("opus 4.6 should skip mitigation, got %q", out)
	}
}

func TestMapReadToolResultToAssistantText_compactKillswitch(t *testing.T) {
	t.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", "1")
	raw := `{"type":"text","file":{"filePath":"x","content":"a","numLines":1,"startLine":1,"totalLines":1}}`
	out, err := MapReadToolResultToAssistantText(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "     1→a") {
		t.Fatalf("expected padded-arrow prefix, got %q", out)
	}
}

func TestMapReadToolResultToAssistantText_memoryFreshness(t *testing.T) {
	t.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", "1")
	raw := `{"type":"text","file":{"filePath":"m.md","content":"x","numLines":1,"startLine":1,"totalLines":1}}`
	old := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	out, err := MapReadToolResultToAssistantText(raw, &ReadToolResultMapOpts{
		MemoryFileMtimeMs: &old,
		Now:               time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "<system-reminder>This memory is ") {
		t.Fatalf("expected memory freshness prefix, got %q", out)
	}
}

func TestReadToolResultMapOptsForToolInput_resolvesMemPath(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	memRoot := t.TempDir()
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memRoot)
	cwd := t.TempDir()
	p := filepath.Join(memRoot, "MEMORY.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]string{"file_path": p})
	opts := ReadToolResultMapOptsForToolInput(b, []string{memRoot}, cwd, "m")
	if opts.MemoryFileMtimeMs == nil {
		t.Fatal("expected memory mtime for path under auto-memory root")
	}
}
