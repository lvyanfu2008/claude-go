package query

import (
	"strings"
	"testing"

	"goc/anthropicmessages"
)

func TestHandleChunk_reasoningFromChoiceMessage(t *testing.T) {
	var saw string
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	chunk := []byte(
		`{"choices":[{"index":0,"delta":{},"message":{"reasoning_content":"from message object"},"finish_reason":null}]}`,
	)
	err := ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
		if ev.Type == "content_block_delta" && strings.Contains(string(ev.Raw), "from message object") {
			saw = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saw == "" {
		t.Fatal("expected message-sourced reasoning_content to emit a thinking_delta")
	}
}
