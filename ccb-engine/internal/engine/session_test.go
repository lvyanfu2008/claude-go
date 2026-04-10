package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/llm"
)

func TestSessionRunTurn_MockAPI_EndTurn(t *testing.T) {
	assistantBody := map[string]any{
		"id":   "msg_1",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{"type": "text", "text": "hello from mock"},
		},
		"stop_reason": "end_turn",
		"usage":       map[string]any{"input_tokens": 1, "output_tokens": 2},
	}
	raw, _ := json.Marshal(assistantBody)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := &anthropic.Client{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		HTTP:    srv.Client(),
		Model:   "claude-3-5-haiku-20241022",
	}

	sess := NewSession(nil)
	sess.AppendUserText("hi")
	ctx := context.Background()
	err := sess.RunTurn(ctx, &llm.AnthropicAdapter{Client: client}, nil, "", StubRunner{}, false)
	if err != nil {
		t.Fatal(err)
	}
	sess.mu.Lock()
	n := len(sess.messages)
	sess.mu.Unlock()
	if n != 2 {
		t.Fatalf("messages len=%d want 2 (user+assistant)", n)
	}
}

func TestSessionRunTurn_MockAPI_ToolUseThenEnd(t *testing.T) {
	first := map[string]any{
		"id":          "msg_tool",
		"type":        "message",
		"role":        "assistant",
		"stop_reason": "tool_use",
		"usage":       map[string]any{"input_tokens": 3, "output_tokens": 4},
		"content": []map[string]any{
			{"type": "tool_use", "id": "tu_1", "name": "echo_stub", "input": map[string]any{"message": "x"}},
		},
	}
	second := map[string]any{
		"id":          "msg_end",
		"type":        "message",
		"role":        "assistant",
		"stop_reason": "end_turn",
		"usage":       map[string]any{"input_tokens": 5, "output_tokens": 6},
		"content": []map[string]any{
			{"type": "text", "text": "done"},
		},
	}
	b1, _ := json.Marshal(first)
	b2, _ := json.Marshal(second)
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		calls++
		if calls == 1 {
			_, _ = w.Write(b1)
			return
		}
		_, _ = w.Write(b2)
	}))
	defer srv.Close()

	client := &anthropic.Client{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		HTTP:    srv.Client(),
		Model:   "claude-3-5-haiku-20241022",
	}

	sess := NewSession(nil)
	sess.AppendUserText("use tool")
	ctx := context.Background()
	err := sess.RunTurn(ctx, &llm.AnthropicAdapter{Client: client}, anthropic.DefaultStubTools(), "", StubRunner{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("API calls=%d want 2", calls)
	}
	sess.mu.Lock()
	n := len(sess.messages)
	sess.mu.Unlock()
	// user, assistant+tools, user+tool_results, assistant
	if n != 4 {
		t.Fatalf("messages len=%d want 4", n)
	}
}

func TestSessionHydrateFromMessages(t *testing.T) {
	sess := NewSession(nil)
	sess.HydrateFromMessages([]anthropic.Message{
		{Role: "user", Content: "prior"},
		{Role: "assistant", Content: "ok"},
	})
	sess.AppendUserText("new")
	sess.mu.Lock()
	n := len(sess.messages)
	first := sess.messages[0].Content
	last := sess.messages[n-1].Content
	sess.mu.Unlock()
	if n != 3 {
		t.Fatalf("len=%d want 3", n)
	}
	if first != "prior" {
		t.Fatalf("first content %v", first)
	}
	if last != "new" {
		t.Fatalf("last content %v", last)
	}
}
