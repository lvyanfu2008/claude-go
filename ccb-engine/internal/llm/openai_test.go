package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func TestOpenAICompat_MockChatCompletions(t *testing.T) {
	body := map[string]any{
		"choices": []map[string]any{
			{
				"finish_reason": "stop",
				"message": map[string]any{
					"role":    "assistant",
					"content": "hello openai",
				},
			},
		},
		"usage": map[string]any{"prompt_tokens": 3, "completion_tokens": 4},
	}
	raw, _ := json.Marshal(body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	o := &OpenAICompat{
		APIKey:  "sk-test",
		BaseURL: srv.URL + "/v1",
		HTTP:    srv.Client(),
		Model:   "deepseek-chat",
	}
	msgs := []anthropic.Message{{Role: "user", Content: "hi"}}
	res, err := o.Complete(context.Background(), msgs, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != "end_turn" {
		t.Fatalf("stop %q", res.StopReason)
	}
	if len(res.Blocks) != 1 || res.Blocks[0].Text != "hello openai" {
		t.Fatalf("blocks %+v", res.Blocks)
	}
}

func TestMessagesToOpenAI_assistantNonOpenAIBlocksHaveContent(t *testing.T) {
	msgs := []anthropic.Message{
		{
			Role: "assistant",
			Content: []anthropic.ContentBlock{{
				Type: "server_tool_use",
				ID:   "srv1",
				Name: "example_server_tool",
				Input: json.RawMessage(`{"q":"x"}`),
			}},
		},
	}
	oa, err := messagesToOpenAI(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(oa) != 1 {
		t.Fatalf("len=%d %+v", len(oa), oa)
	}
	if oa[0].Role != "assistant" {
		t.Fatalf("role=%q", oa[0].Role)
	}
	s, ok := oa[0].Content.(string)
	if !ok || s == "" {
		t.Fatalf("want non-empty string content, got %#v", oa[0].Content)
	}
	if len(oa[0].ToolCalls) != 0 {
		t.Fatalf("unexpected tool_calls %+v", oa[0].ToolCalls)
	}
}

func TestMessagesToOpenAI_userTextBlocksNotDropped(t *testing.T) {
	msgs := []anthropic.Message{
		{
			Role: "user",
			Content: []anthropic.ContentBlock{
				{Type: "text", Text: "你好"},
			},
		},
	}
	oa, err := messagesToOpenAI(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(oa) != 1 || oa[0].Role != "user" {
		t.Fatalf("%+v", oa)
	}
	if oa[0].Content != "你好" {
		t.Fatalf("content=%#v", oa[0].Content)
	}
}
