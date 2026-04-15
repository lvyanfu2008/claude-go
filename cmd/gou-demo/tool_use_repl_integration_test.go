package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/ccb-engine/skilltools"
	"goc/conversation-runtime/streamingtool"
	"goc/toolexecution"
	"goc/types"
)

// Pipeline: assistant REPL tool_use → RunToolUseChan → user tool_result (TS runToolUse order).
func TestRunToolUseChan_replWithParityRunnerInGouDemoStack(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(p, []byte("pipeline"), 0o644); err != nil {
		t.Fatal(err)
	}
	replInput, err := json.Marshal(map[string]any{
		"tool": "Read",
		"input": map[string]any{"file_path": p},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := &skilltools.ParityToolRunner{
		DemoToolRunner:   skilltools.DemoToolRunner{},
		WorkDir:          dir,
		LocalBashDefault: true,
	}
	ch := toolexecution.RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "repl-tu-e2e", Name: "REPL", Input: replInput},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst1"},
		toolexecution.ExecutionDeps{
			RandomUUID: func() string { return "usr1" },
			InvokeTool: runner.Run,
		},
		nil,
	)
	var got *types.Message
	for u := range ch {
		if u.Message != nil {
			got = u.Message
		}
	}
	if got == nil || got.Type != types.MessageTypeUser {
		t.Fatalf("expected user tool_result, got %+v", got)
	}
	body := string(got.Message) + string(got.Content) + string(got.ToolUseResult)
	if !strings.Contains(body, "pipeline") || !strings.Contains(body, "repl-tu-e2e") {
		t.Fatalf("unexpected tool_result (message/content/toolUseResult): %s", body)
	}
}
