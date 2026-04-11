package toolexecution

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

func TestRunToolUseChan_unknownTool(t *testing.T) {
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "tu1", Name: "Missing", Input: []byte(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "asst1"},
		ExecutionDeps{RandomUUID: func() string { return "fixed-uuid" }},
		nil,
	)
	var got int
	for u := range ch {
		if u.Message == nil {
			t.Fatalf("expected message")
		}
		got++
		if got > 1 {
			t.Fatal("too many updates")
		}
		if u.Message.UUID != "fixed-uuid" {
			t.Fatalf("uuid %q", u.Message.UUID)
		}
	}
	if got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestRunToolUseChan_invokeOK(t *testing.T) {
	ch := RunToolUseChan(context.Background(),
		streamingtool.ToolUseBlock{ID: "x", Name: "Echo", Input: []byte(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "a"},
		ExecutionDeps{
			RandomUUID: func() string { return "u1" },
			InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
				return `{"ok":true}`, false, nil
			},
		},
		nil,
	)
	for u := range ch {
		if u.Message == nil || u.Message.Type != types.MessageTypeUser {
			t.Fatalf("%+v", u.Message)
		}
	}
}

func TestRunToolUseChan_abortDuringInvoke(t *testing.T) {
	toolAbort := streamingtool.NewAbortController()
	ctx := context.Background()
	ch := RunToolUseChan(ctx,
		streamingtool.ToolUseBlock{ID: "x", Name: "Slow", Input: []byte(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "a"},
		ExecutionDeps{
			RandomUUID: func() string { return "u-abort" },
			InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
				toolAbort.Abort("interrupt")
				select {
				case <-ctx.Done():
					return "", false, ctx.Err()
				case <-time.After(200 * time.Millisecond):
				}
				return "late", false, nil
			},
		},
		toolAbort,
	)
	var n int
	for range ch {
		n++
	}
	if n < 1 {
		t.Fatalf("expected at least one yield, got %d", n)
	}
}
