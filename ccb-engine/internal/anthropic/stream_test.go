package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateMessageStream_httptest(t *testing.T) {
	sse := "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-3-5-haiku-20241022\",\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":3,\"output_tokens\":0}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"usage\":{\"output_tokens\":2}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("accept") != "text/event-stream" {
			t.Errorf("accept header: %q", r.Header.Get("accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, sse)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	c := &Client{
		APIKey:  "test-key",
		BaseURL: strings.TrimSuffix(srv.URL, ""),
		HTTP:    srv.Client(),
		Model:   "claude-3-5-haiku-20241022",
	}

	var types []string
	err := c.CreateMessageStream(context.Background(), CreateMessageRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(ev MessageStreamEvent) error {
		types = append(types, ev.Type)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_delta", "message_stop"}
	if len(types) != len(want) {
		t.Fatalf("types=%v want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("at %d: got %q want %q", i, types[i], want[i])
		}
	}
}
