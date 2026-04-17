package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/socketserve"
	"goc/ccb-engine/llmturn"
)

type textOnlyCompleter struct{}

func (textOnlyCompleter) Complete(
	_ context.Context,
	_ []anthropic.Message,
	_ []anthropic.ToolDefinition,
	_ string,
) (*llmturn.TurnResult, error) {
	return &llmturn.TurnResult{
		Blocks: []anthropic.ContentBlock{
			{Type: "text", Text: "hello"},
		},
		StopReason: "end_turn",
	}, nil
}

func TestServeConnTextOnly(t *testing.T) {
	t.Parallel()
	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	done := make(chan struct{})
	go func() {
		socketserve.HandleConn(srv, textOnlyCompleter{}, anthropic.DefaultStubTools())
		close(done)
	}()

	_ = cli.SetDeadline(time.Now().Add(10 * time.Second))

	req := map[string]any{
		"method": "SubmitUserTurn",
		"id":     "req-text-1",
		"payload": map[string]any{
			"text": "hi",
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Write(append(body, '\n')); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(cli)
	var sawDelta bool
	var sawEnd bool
	for !sawEnd {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		line = strings.TrimSpace(line)
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		switch ev["type"] {
		case "assistant_delta":
			sawDelta = true
		case "response_end":
			sawEnd = true
		}
	}
	if !sawDelta {
		t.Fatal("expected assistant_delta")
	}
	_ = cli.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("HandleConn did not exit")
	}
}

// seqMockCompleter drives two-step tool_use then end_turn for serve integration.
type seqMockCompleter struct {
	mu    sync.Mutex
	calls int
}

func (m *seqMockCompleter) Complete(
	_ context.Context,
	_ []anthropic.Message,
	_ []anthropic.ToolDefinition,
	_ string,
) (*llmturn.TurnResult, error) {
	m.mu.Lock()
	m.calls++
	c := m.calls
	m.mu.Unlock()

	if c == 1 {
		return &llmturn.TurnResult{
			Blocks: []anthropic.ContentBlock{
				{
					Type:  "tool_use",
					ID:    "tu_echo",
					Name:  "echo_stub",
					Input: json.RawMessage(`{"message":"z"}`),
				},
			},
			StopReason: "tool_use",
		}, nil
	}
	return &llmturn.TurnResult{
		Blocks: []anthropic.ContentBlock{
			{Type: "text", Text: "after_tool"},
		},
		StopReason: "end_turn",
	}, nil
}

func TestServeConnSubmitTurnToolBridgeRoundTrip(t *testing.T) {
	m := &seqMockCompleter{}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		srv, err := ln.Accept()
		if err != nil {
			t.Error(err)
			close(done)
			return
		}
		socketserve.HandleConn(srv, m, anthropic.DefaultStubTools())
		close(done)
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	_ = cli.SetDeadline(time.Now().Add(10 * time.Second))

	req := map[string]any{
		"method": "SubmitUserTurn",
		"id":     "req-bridge-1",
		"payload": map[string]any{
			"text": "hi",
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Write(append(body, '\n')); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(cli)
	sawExecute := false
	sawResponseEnd := false
readLoop:
	for !sawResponseEnd {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		line = strings.TrimSpace(line)
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		typ, _ := ev["type"].(string)
		switch typ {
		case "execute_tool":
			sawExecute = true
			callID, _ := ev["call_id"].(string)
			if callID == "" {
				t.Fatalf("execute_tool missing call_id: %s", line)
			}
			tr, err := json.Marshal(map[string]any{
				"call_id":     callID,
				"tool_use_id": "tu_echo",
				"content":     "from_test_client",
				"is_error":    false,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cli.Write(append(tr, '\n')); err != nil {
				t.Fatal(err)
			}
		case "response_end":
			sawResponseEnd = true
			break readLoop
		}
	}
	if !sawExecute {
		t.Fatal("never saw execute_tool")
	}
	if !sawResponseEnd {
		t.Fatal("never saw response_end")
	}

	_ = cli.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("HandleConn did not exit")
	}
}
