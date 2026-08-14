package query

import (
	"encoding/json"
	"testing"
)

func TestAnthropicWireMessagesToOpenAI_ReasoningContentWhenThinkingEnabled(t *testing.T) {
	// deepseek-v4-pro has thinking on by default (do not set OPENAI_ENABLE_THINKING=0).
	msgs := []byte(`[
  {"role":"user","content":[{"type":"text","text":"hi"}]},
  {"role":"assistant","content":[
    {"type":"thinking","thinking":"Let me reason."},
    {"type":"text","text":"Hello."}
  ]}
]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 2 {
		t.Fatalf("expected user+assistant, got %d", len(out))
	}
	last := out[len(out)-1]
	if rc, _ := last["reasoning_content"].(string); rc != "Let me reason." {
		t.Fatalf("reasoning_content: %#v", last["reasoning_content"])
	}
}

func TestAnthropicWireMessagesToOpenAI_ThinkingOmittedWhenThinkingDisabled(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "0")
	// No thinking blocks in content — gpt-4o should not get reasoning_content.
	msgs := []byte(`[
  {"role":"assistant","content":[
    {"type":"text","text":"out"}
  ]}
]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	a := out[len(out)-1]
	if _, ok := a["reasoning_content"]; ok {
		t.Fatalf("did not want reasoning_content on gpt-4o with text-only: %#v", a)
	}
}

func TestAnthropicWireMessagesToOpenAI_ReplayThinkingWhenModelHeuristicOff(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "0")
	// API replay: prior turn has thinking; must map to reasoning_content even if resolved id is gpt-4o
	// and IsOpenAIThinkingEnabled is false (per convertMessages.ts fromContent guard).
	msgs := []byte(`[
  {"role":"assistant","content":[
    {"type":"thinking","thinking":"internal"},
    {"type":"text","text":"out"}
  ]}
]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	a := out[len(out)-1]
	if rc, _ := a["reasoning_content"].(string); rc != "internal" {
		t.Fatalf("reasoning_content: %#v", a["reasoning_content"])
	}
}

func TestAnthropicWireMessagesToOpenAI_RedactedThinkingAsReasoning(t *testing.T) {
	msgs := []byte(`[
  {"role":"assistant","content":[
    {"type":"redacted_thinking","data":"opaque"},
    {"type":"text","text":"ok"}
  ]}
]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	a := out[len(out)-1]
	if rc, _ := a["reasoning_content"].(string); rc != "opaque" {
		t.Fatalf("reasoning_content: %#v", a["reasoning_content"])
	}
}

func TestAnthropicWireMessagesToOpenAI_MultipleThinkingJoins(t *testing.T) {
	msgs := []byte(`[
  {"role":"assistant","content":[
    {"type":"thinking","thinking":"A"},
    {"type":"thinking","thinking":"B"},
    {"type":"text","text":"x"}
  ]}
]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-reasoner")
	if err != nil {
		t.Fatal(err)
	}
	rc, _ := out[0]["reasoning_content"].(string)
	if rc != "A\nB" {
		t.Fatalf("got %q", rc)
	}
}

func TestAnthropicWireMessagesToOpenAI_ReasoningKeyConfigWritesReasoning(t *testing.T) {
	// OPENAI_REASONING_KEY=1 → echo reasoning under the "reasoning" key only.
	t.Setenv("OPENAI_REASONING_KEY", "1")
	msgs := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"hi"}]},
	  {"role":"assistant","content":[
	    {"type":"thinking","thinking":"Let me reason."},
	    {"type":"text","text":"Hello."}
	  ]}
	]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	last := out[len(out)-1]
	if r, _ := last["reasoning"].(string); r != "Let me reason." {
		t.Fatalf("reasoning key must carry the reasoning when configured, got %#v", last["reasoning"])
	}
	if _, has := last["reasoning_content"]; has {
		t.Fatalf("reasoning_content should not be written when configured to reasoning, got %#v", last)
	}
}

func TestAnthropicWireMessagesToOpenAI_ReasoningKeyDefaultWritesContent(t *testing.T) {
	// Default (no env): echo reasoning under reasoning_content only.
	msgs := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"hi"}]},
	  {"role":"assistant","content":[
	    {"type":"thinking","thinking":"Let me reason."},
	    {"type":"text","text":"Hello."}
	  ]}
	]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	last := out[len(out)-1]
	if rc, _ := last["reasoning_content"].(string); rc != "Let me reason." {
		t.Fatalf("reasoning_content must carry reasoning by default, got %#v", last["reasoning_content"])
	}
	if _, has := last["reasoning"]; has {
		t.Fatalf("reasoning should not be written by default, got %#v", last)
	}
}

func TestAnthropicWireMessagesToOpenAI_ThinkingModeEmitsEmptyReasoning(t *testing.T) {
	// Thinking mode on: an assistant turn with NO thinking block must still
	// carry the reasoning key (empty string), so the model knows to emit
	// chain-of-thought on the next request.
	msgs := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"hi"}]},
	  {"role":"assistant","content":[
	    {"type":"text","text":"no thinking here"}
	  ]}
	]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	last := out[len(out)-1]
	if _, has := last["reasoning_content"]; !has {
		t.Fatalf("thinking-mode assistant must carry reasoning_content even when empty, got %#v", last)
	}
	if rc, _ := last["reasoning_content"].(string); rc != "" {
		t.Fatalf("expected empty reasoning_content, got %q", rc)
	}
}

func TestAnthropicWireMessagesToOpenAI_NonThinkingModeNoEmptyReasoning(t *testing.T) {
	// Non-thinking model: assistant turn with no thinking block must NOT get a
	// reasoning key at all.
	t.Setenv("OPENAI_ENABLE_THINKING", "0")
	msgs := []byte(`[
	  {"role":"assistant","content":[
	    {"type":"text","text":"no thinking here"}
	  ]}
	]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	last := out[len(out)-1]
	if _, has := last["reasoning_content"]; has {
		t.Fatalf("non-thinking assistant must not get reasoning_content, got %#v", last)
	}
}

func TestAnthropicWireMessagesToOpenAI_ThinkingModeEmptyRespectsKeyConfig(t *testing.T) {
	// OPENAI_REASONING_KEY=1 + thinking mode + no thinking block → empty
	// "reasoning" key (not reasoning_content).
	t.Setenv("OPENAI_REASONING_KEY", "1")
	msgs := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"hi"}]},
	  {"role":"assistant","content":[
	    {"type":"text","text":"no thinking here"}
	  ]}
	]`)
	out, err := anthropicWireMessagesToOpenAI(json.RawMessage(msgs), nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	last := out[len(out)-1]
	if r, _ := last["reasoning"].(string); r != "" {
		t.Fatalf("expected empty reasoning key, got %#v", last["reasoning"])
	}
	if _, has := last["reasoning_content"]; has {
		t.Fatalf("reasoning_content should not be present when configured to reasoning, got %#v", last)
	}
}
