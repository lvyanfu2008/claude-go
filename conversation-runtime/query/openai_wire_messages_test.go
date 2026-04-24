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
	if _, ok := a["reasoning_content"]; ok {
		t.Fatalf("did not want reasoning_content on gpt-4o: %#v", a)
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
