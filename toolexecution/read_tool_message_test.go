package toolexecution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/ccb-engine/localtools"
	"goc/types"
)

func TestSyntheticToolMessageAfterInvoke_readSeparatesContentAndToolUseResult(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.go")
	src := "package main\n\nfunc F() {}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(map[string]any{
		"file_path": p,
		"offset":    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawOut, ierr, err := localtools.ReadFromJSON(input, []string{dir}, localtools.NewReadFileState(), nil)
	if err != nil || ierr {
		t.Fatalf("ReadFromJSON: err=%v isErr=%v out=%q", err, ierr, rawOut)
	}
	deps := ExecutionDeps{
		RandomUUID:    func() string { return "fixed-read-msg-uuid" },
		MainLoopModel: "claude-sonnet-4-20250514",
		ReadToolRoots: []string{dir},
		ReadToolMemCWD: dir,
	}
	msg := syntheticToolMessageAfterInvoke(deps, "Read", "tu-read-1", input, rawOut, false, "asst-1")
	if msg.Type != types.MessageTypeUser {
		t.Fatalf("type %s", msg.Type)
	}
	var inner struct {
		Role    string `json:"role"`
		Content []struct {
			Type       string `json:"type"`
			Content    any    `json:"content"`
			ToolUseID  string `json:"tool_use_id"`
			IsError    bool   `json:"is_error"`
		} `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &inner); err != nil {
		t.Fatal(err)
	}
	if len(inner.Content) != 1 || inner.Content[0].Type != "tool_result" {
		t.Fatalf("content blocks: %+v", inner.Content)
	}
	trContent, ok := inner.Content[0].Content.(string)
	if !ok || !strings.Contains(trContent, "1\tpackage main") {
		t.Fatalf("tool_result.content want numbered text, got %#v", inner.Content[0].Content)
	}
	var tur struct {
		Type string `json:"type"`
		File struct {
			FilePath string `json:"filePath"`
			Content  string `json:"content"`
		} `json:"file"`
	}
	if err := json.Unmarshal(msg.ToolUseResult, &tur); err != nil {
		t.Fatalf("toolUseResult JSON: %v raw=%s", err, string(msg.ToolUseResult))
	}
	if tur.Type != "text" || tur.File.Content != src {
		t.Fatalf("toolUseResult: %+v", tur)
	}
}
