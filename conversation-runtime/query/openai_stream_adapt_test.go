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

func TestHandleChunk_reasoningFromDeltaWhenConfigured(t *testing.T) {
	// With OPENAI_REASONING_KEY=1 the chain-of-thought is read from delta.reasoning.
	t.Setenv("OPENAI_REASONING_KEY", "1")
	var saw string
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	chunk := []byte(
		`{"choices":[{"index":0,"delta":{"reasoning":"reasoning in delta"},"finish_reason":null}]}`,
	)
	err := ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
		if ev.Type == "content_block_delta" && strings.Contains(string(ev.Raw), "reasoning in delta") {
			saw = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saw == "" {
		t.Fatal("expected delta-sourced reasoning to emit a thinking_delta when configured")
	}
}

func TestHandleChunk_reasoningFromChoiceMessageWhenConfigured(t *testing.T) {
	// With OPENAI_REASONING_KEY=1 the message key is read as reasoning.
	t.Setenv("OPENAI_REASONING_KEY", "1")
	var saw string
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	chunk := []byte(
		`{"choices":[{"index":0,"delta":{},"message":{"reasoning":"from message reasoning key"},"finish_reason":null}]}`,
	)
	err := ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
		if ev.Type == "content_block_delta" && strings.Contains(string(ev.Raw), "from message reasoning key") {
			saw = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saw == "" {
		t.Fatal("expected message-sourced reasoning to emit a thinking_delta when configured")
	}
}

func TestReplayOpenAINonStreamReasoningWhenConfigured(t *testing.T) {
	// With OPENAI_REASONING_KEY=1, non-stream message.reasoning must emit a thinking block.
	t.Setenv("OPENAI_REASONING_KEY", "1")
	body := []byte(`{
	  "choices": [{
	    "message": {"role": "assistant", "content": "answer", "reasoning": "non-stream reasoning"},
	    "finish_reason": "stop"
	  }]
	}`)
	var sawThinking string
	err := ReplayOpenAINonStreamChatResponse(body, "deepseek-v4", func(ev anthropicmessages.MessageStreamEvent) error {
		if strings.Contains(string(ev.Raw), "non-stream reasoning") {
			sawThinking = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sawThinking == "" {
		t.Fatal("expected non-stream reasoning to emit a thinking block when configured")
	}
}

func TestHandleChunk_defaultIgnoresReasoningKey(t *testing.T) {
	// Default config reads reasoning_content only: a delta carrying ONLY
	// reasoning must NOT produce a thinking block (no fallback between keys).
	var saw string
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	chunk := []byte(
		`{"choices":[{"index":0,"delta":{"reasoning":"should be ignored by default"},"finish_reason":null}]}`,
	)
	err := ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
		if ev.Type == "content_block_delta" && strings.Contains(string(ev.Raw), "should be ignored by default") {
			saw = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saw != "" {
		t.Fatal("default config must ignore the reasoning key (no cross-key fallback)")
	}
}

func TestHandleChunk_ReasoningKeyConfigPrefersReasoning(t *testing.T) {
	// With OPENAI_REASONING_KEY=1, a chunk carrying BOTH keys must prefer the
	// "reasoning" value for the thinking block.
	t.Setenv("OPENAI_REASONING_KEY", "1")
	var saw string
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	chunk := []byte(
		`{"choices":[{"index":0,"delta":{"reasoning_content":"content key","reasoning":"reasoning key preferred"},"finish_reason":null}]}`,
	)
	err := ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
		if ev.Type == "content_block_delta" && strings.Contains(string(ev.Raw), "reasoning key preferred") {
			saw = string(ev.Raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saw == "" {
		t.Fatal("expected the configured 'reasoning' key value to be emitted")
	}
}
