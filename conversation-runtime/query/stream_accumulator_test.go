package query

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/anthropicmessages"
)

// sseTwoTextBlocks is two sequential text blocks (indexes 0 and 1) before end_turn.
func sseTwoTextBlocks() string {
	return "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":1}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"usage\":{\"output_tokens\":2}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
}

func TestAssistantStreamAccumulator_multiTextBlocks(t *testing.T) {
	acc := newAssistantStreamAccumulator()
	err := anthropicmessages.ReadSSE(strings.NewReader(sseTwoTextBlocks()), func(data []byte) error {
		return anthropicmessages.ProcessStreamPayloads(data, acc.OnEvent)
	})
	if err != nil {
		t.Fatal(err)
	}
	inner, err := acc.AssistantWire("asst-1")
	if err != nil {
		t.Fatal(err)
	}
	var wrap struct {
		Role    string `json:"role"`
		ID      string `json:"id"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(inner, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.ID != "msg_1" {
		t.Fatalf("id=%q want msg_1", wrap.ID)
	}
	if wrap.Role != "assistant" || len(wrap.Content) != 2 {
		t.Fatalf("role=%q content=%d", wrap.Role, len(wrap.Content))
	}
	if wrap.Content[0].Type != "text" || wrap.Content[0].Text != "hello" {
		t.Fatalf("block0=%+v", wrap.Content[0])
	}
	if wrap.Content[1].Type != "text" || wrap.Content[1].Text != " world" {
		t.Fatalf("block1=%+v", wrap.Content[1])
	}
	if acc.StopReason() != "end_turn" {
		t.Fatalf("stop=%q", acc.StopReason())
	}
}
