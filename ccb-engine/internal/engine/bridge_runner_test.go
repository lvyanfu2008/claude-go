package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"goc/ccb-engine/internal/protocol"
)

func TestBridgeRunnerDeliversToolResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	writeCh := make(chan protocol.StreamEvent, 1)
	var writeMu sync.Mutex
	br := NewBridgeRunner(ctx, &writeMu, func(ev protocol.StreamEvent) error {
		writeCh <- ev
		return nil
	})
	br.SetStateRevProvider(func() uint64 { return 7 })
	br.SetToolExecutePolicy(map[string]any{"decision": "allow", "source": "ccb-engine"})

	type runOut struct {
		s   string
		ie  bool
		err error
	}
	out := make(chan runOut, 1)
	go func() {
		s, ie, e := br.Run(ctx, "Read", "toolu_abc", json.RawMessage(`{"path":"x"}`))
		out <- runOut{s, ie, e}
	}()

	select {
	case ev := <-writeCh:
		if ev.Type != "execute_tool" {
			t.Fatalf("want execute_tool, got %q", ev.Type)
		}
		if ev.CallID == "" || ev.ToolUseID != "toolu_abc" || ev.Name != "Read" {
			t.Fatalf("unexpected event: %+v", ev)
		}
		if ev.StateRev != 7 {
			t.Fatalf("state_rev: got %d want 7", ev.StateRev)
		}
		if ev.Policy == nil || ev.Policy["decision"] != "allow" {
			t.Fatalf("expected policy.decision allow, got %+v", ev.Policy)
		}
		if !br.DeliverToolResult(ev.CallID, ev.ToolUseID, "done", false) {
			t.Fatal("DeliverToolResult returned false")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for execute_tool")
	}

	select {
	case r := <-out:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.ie {
			t.Fatal("unexpected is_error")
		}
		if r.s != "done" {
			t.Fatalf("content: got %q", r.s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Run")
	}
}

func TestBridgeRunnerParentContextCancel(t *testing.T) {
	t.Parallel()
	parent, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	var writeMu sync.Mutex
	br := NewBridgeRunner(parent, &writeMu, func(protocol.StreamEvent) error {
		close(started)
		return nil
	})

	errCh := make(chan error, 1)
	go func() {
		_, _, e := br.Run(context.Background(), "N", "id", json.RawMessage("{}"))
		errCh <- e
	}()

	<-started
	cancel()

	select {
	case e := <-errCh:
		if e == nil {
			t.Fatal("expected error from cancelled bridge context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Run after cancel")
	}
}
