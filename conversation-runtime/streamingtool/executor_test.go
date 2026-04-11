package streamingtool_test

import (
	"context"
	"testing"

	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

type fakeTool struct {
	name   string
	conc   bool
	cancel bool
}

func (f *fakeTool) Name() string { return f.name }

func (f *fakeTool) InputOK([]byte) (any, bool) { return struct{}{}, true }

func (f *fakeTool) IsConcurrencySafe(any) bool { return f.conc }

func (f *fakeTool) InterruptBehavior() string {
	if f.cancel {
		return "cancel"
	}
	return "block"
}

type fakeCtx struct {
	query     *streamingtool.AbortController
	inProg    map[string]struct{}
	interrupt bool
}

func (f *fakeCtx) QueryAbort() *streamingtool.AbortController { return f.query }

func (f *fakeCtx) SetInProgressToolUseIDs(updater func(map[string]struct{}) map[string]struct{}) {
	f.inProg = updater(f.inProg)
}

func (f *fakeCtx) SetHasInterruptibleToolInProgress(v bool) { f.interrupt = v }

type fakeRunner struct {
	ch chan streamingtool.ToolRunUpdate
}

func (r *fakeRunner) RunToolUpdates(
	block streamingtool.ToolUseBlock,
	assistant types.Message,
	canUseTool any,
	tcx streamingtool.ToolUseContextPort,
	toolAbort *streamingtool.AbortController,
) <-chan streamingtool.ToolRunUpdate {
	_ = assistant
	_ = canUseTool
	_ = tcx
	_ = toolAbort
	if r.ch != nil {
		return r.ch
	}
	ch := make(chan streamingtool.ToolRunUpdate, 1)
	um := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "u1",
		Message: []byte(`{"role":"user","content":[{"type":"tool_result","tool_use_id":"` + block.ID + `","content":"ok"}]}`),
	}
	ch <- streamingtool.ToolRunUpdate{Message: &um}
	close(ch)
	return ch
}

func TestStreamingToolExecutor_unknownTool(t *testing.T) {
	q := streamingtool.NewAbortController()
	ctx := &fakeCtx{query: q, inProg: map[string]struct{}{}}
	ex := streamingtool.NewStreamingToolExecutor(
		func(name string) (streamingtool.ToolBehavior, bool) { return nil, false },
		nil,
		ctx,
		&fakeRunner{},
	)
	ex.AddTool(streamingtool.ToolUseBlock{ID: "x1", Name: "Nope", Input: []byte(`{}`)}, types.Message{Type: types.MessageTypeAssistant, UUID: "a1"})
	var got int
	for range ex.GetCompletedResults() {
		got++
	}
	if got != 1 {
		t.Fatalf("want 1 synthetic result, got %d", got)
	}
}

func TestStreamingToolExecutor_remainingResults(t *testing.T) {
	q := streamingtool.NewAbortController()
	ctx := &fakeCtx{query: q, inProg: map[string]struct{}{}}
	r := &fakeRunner{}
	ex := streamingtool.NewStreamingToolExecutor(
		func(name string) (streamingtool.ToolBehavior, bool) {
			if name == "Read" {
				return &fakeTool{name: "Read", conc: true}, true
			}
			return nil, false
		},
		nil,
		ctx,
		r,
	)
	ex.AddTool(streamingtool.ToolUseBlock{ID: "t1", Name: "Read", Input: []byte(`{"file_path":"/x"}`)}, types.Message{Type: types.MessageTypeAssistant, UUID: "a1"})
	var n int
	for u, err := range ex.RemainingResults(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if u.Message != nil {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want 1 message, got %d", n)
	}
}
