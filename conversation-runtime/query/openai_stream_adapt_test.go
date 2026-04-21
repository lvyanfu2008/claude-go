package query

import (
	"bytes"
	"encoding/json"
	"testing"

	"goc/anthropicmessages"
)

func TestOpenAIStreamAdapter_textOnly(t *testing.T) {
	ad := newOpenAIStreamAdapter("deepseek-chat")
	var types []string
	emit := func(ev anthropicmessages.MessageStreamEvent) error {
		types = append(types, ev.Type)
		return nil
	}
	chunks := []string{
		`{"choices":[{"index":0,"delta":{}}],"model":"deepseek-chat"}`,
		`{"choices":[{"index":0,"delta":{"content":"hello"}}]}`,
		`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	}
	for _, c := range chunks {
		if err := ad.HandleChunk([]byte(c), emit); err != nil {
			t.Fatal(err)
		}
	}
	if err := ad.FlushOpenBlocks(emit); err != nil {
		t.Fatal(err)
	}
	if len(types) < 2 {
		t.Fatalf("events: %v", types)
	}
	if types[0] != "message_start" {
		t.Fatalf("first type %q", types[0])
	}
}

func TestOpenAIStreamAdapter_toolCalls_argumentsAsObject(t *testing.T) {
	ad := newOpenAIStreamAdapter("test-model")
	var events []anthropicmessages.MessageStreamEvent
	emit := func(ev anthropicmessages.MessageStreamEvent) error {
		events = append(events, ev)
		return nil
	}
	chunk := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_x","function":{"name":"Bash","arguments":{"command":"pwd"}}}]}}]}`
	if err := ad.HandleChunk([]byte(chunk), emit); err != nil {
		t.Fatal(err)
	}
	if err := ad.HandleChunk([]byte(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`), emit); err != nil {
		t.Fatal(err)
	}
	if err := ad.FlushOpenBlocks(emit); err != nil {
		t.Fatal(err)
	}
	var jsonDeltas int
	for _, ev := range events {
		if ev.Type == "content_block_delta" && bytes.Contains(ev.Raw, []byte(`"input_json_delta"`)) {
			jsonDeltas++
		}
	}
	if jsonDeltas == 0 {
		t.Fatalf("expected input_json_delta events, got %d events types=%v", len(events), eventTypes(events))
	}
	acc := newAssistantStreamAccumulator()
	for _, ev := range events {
		if err := acc.OnEvent(ev); err != nil {
			t.Fatal(err)
		}
	}
	blocks := acc.ToolUseBlocks()
	if len(blocks) != 1 {
		t.Fatalf("want 1 tool block, got %d", len(blocks))
	}
	if blocks[0].Name != "Bash" {
		t.Fatalf("name %q", blocks[0].Name)
	}
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(blocks[0].Input, &in); err != nil {
		t.Fatal(err)
	}
	if in.Command != "pwd" {
		t.Fatalf("command %q want pwd", in.Command)
	}
}

func eventTypes(events []anthropicmessages.MessageStreamEvent) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.Type)
	}
	return out
}

func TestNormalizeOpenAINonStreamChatBodyToolCallsLoose_flatArgsThenReplay(t *testing.T) {
	body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"name":"demo","arguments":{"path":"/tmp"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`)
	fixed := NormalizeOpenAINonStreamChatBodyToolCallsLoose(body)
	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAINonStreamChatResponse(fixed, "test-model", acc.OnEvent); err != nil {
		t.Fatal(err)
	}
	if !acc.HasToolUse() {
		t.Fatal("expected tool use after normalize+replay")
	}
	blocks := acc.ToolUseBlocks()
	if len(blocks) != 1 || blocks[0].Name != "demo" {
		t.Fatalf("blocks=%+v", blocks)
	}
}

func TestReplayOpenAINonStreamChatResponse_textOnly(t *testing.T) {
	body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"你好"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`)
	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAINonStreamChatResponse(body, "deepseek-chat", acc.OnEvent); err != nil {
		t.Fatal(err)
	}
	wire, err := acc.AssistantWire("u1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("你好")) {
		t.Fatalf("wire: %s", string(wire))
	}
	if acc.HasToolUse() {
		t.Fatal("unexpected tool use")
	}
}

func TestReplayOpenAIStreamChatResponse_reasoningAndText_deepseekReasoner(t *testing.T) {
	sse := "" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{}}],\"model\":\"deepseek-reasoner\"}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"think\\n\"}}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"answer\"}}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAIStreamChatResponse([]byte(sse), "deepseek-reasoner", acc.OnEvent); err != nil {
		t.Fatal(err)
	}
	wire, err := acc.AssistantWire("u1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("think")) {
		t.Fatalf("expected thinking in wire, got %s", string(wire))
	}
	if !bytes.Contains(wire, []byte("answer")) {
		t.Fatalf("expected answer text in wire, got %s", string(wire))
	}
	if acc.HasToolUse() {
		t.Fatal("unexpected tool use")
	}
}

func TestReplayOpenAINonStreamChatResponse_reasoningAndText_deepseekReasoner(t *testing.T) {
	body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","reasoning_content":"think step\n","content":"answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`)
	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAINonStreamChatResponse(body, "deepseek-reasoner", acc.OnEvent); err != nil {
		t.Fatal(err)
	}
	wire, err := acc.AssistantWire("u1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("think step")) {
		t.Fatalf("expected thinking in wire, got %s", string(wire))
	}
	if !bytes.Contains(wire, []byte("answer")) {
		t.Fatalf("expected answer text in wire, got %s", string(wire))
	}
	if acc.HasToolUse() {
		t.Fatal("unexpected tool use")
	}
}
