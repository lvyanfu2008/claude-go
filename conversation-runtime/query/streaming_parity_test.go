package query

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"goc/anthropicmessages"
	"goc/toolexecution"
	"goc/types"
)

// textOnlySSE is a minimal Anthropic-style stream (one text block, end_turn, message_stop).
func textOnlySSE() string {
	return "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"usage\":{\"output_tokens\":1}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
}

func TestStreamingParity_textOnly(t *testing.T) {
	t.Setenv("GOU_DEMO_STREAMING_TOOL_EXECUTION", "1")
	t.Setenv("GOU_QUERY_STREAMING_PARITY", "")
	tools, _ := json.Marshal([]map[string]any{{"name": "echo_stub", "input_schema": map[string]any{"type": "object"}}})

	var calls int
	deps := ProductionDeps()
	deps.StreamPost = func(ctx context.Context, p anthropicmessages.PostStreamParams) error {
		calls++
		return anthropicmessages.ReadSSE(strings.NewReader(textOnlySSE()), func(data []byte) error {
			return anthropicmessages.ProcessStreamPayloads(data, p.Emit)
		})
	}

	var got []types.MessageType
	ctx := context.Background()
	for y, err := range Query(ctx, QueryParams{
		Messages: []types.Message{{
			Type: types.MessageTypeUser,
			UUID: "u1",
			Message: mustJSON(t, map[string]any{
				"role": "user", "content": "hi",
			}),
		}},
		SystemPrompt: AsSystemPrompt([]string{"sys"}),
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				Tools:         tools,
				MainLoopModel: "claude-3-5-haiku-20241022",
			},
		},
		StreamingParity: true,
		Deps:            &deps,
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if y.Message != nil {
			got = append(got, y.Message.Type)
		}
		if y.Terminal != nil {
			break
		}
	}
	if calls != 1 {
		t.Fatalf("stream calls=%d", calls)
	}
	if len(got) < 1 || got[0] != types.MessageTypeAssistant {
		t.Fatalf("got %#v", got)
	}
}

func TestStreamingParity_OnQueryYield(t *testing.T) {
	t.Setenv("GOU_DEMO_STREAMING_TOOL_EXECUTION", "1")
	tools, _ := json.Marshal([]map[string]any{{"name": "echo_stub", "input_schema": map[string]any{"type": "object"}}})

	var hookCalls int
	deps := ProductionDeps()
	deps.StreamPost = func(ctx context.Context, p anthropicmessages.PostStreamParams) error {
		return anthropicmessages.ReadSSE(strings.NewReader(textOnlySSE()), func(data []byte) error {
			return anthropicmessages.ProcessStreamPayloads(data, p.Emit)
		})
	}
	deps.OnQueryYield = func(ctx context.Context, y QueryYield) error {
		if y.Message != nil {
			hookCalls++
		}
		return nil
	}

	ctx := context.Background()
	for y, err := range Query(ctx, QueryParams{
		Messages: []types.Message{{
			Type: types.MessageTypeUser,
			UUID: "u1",
			Message: mustJSON(t, map[string]any{
				"role": "user", "content": "hi",
			}),
		}},
		SystemPrompt: AsSystemPrompt([]string{"sys"}),
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				Tools:         tools,
				MainLoopModel: "claude-3-5-haiku-20241022",
			},
		},
		StreamingParity: true,
		Deps:            &deps,
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if y.Terminal != nil {
			break
		}
	}
	if hookCalls < 1 {
		t.Fatalf("OnQueryYield calls=%d", hookCalls)
	}
}

func TestStreamingParity_textOnly_gateViaGOUQueryEnv(t *testing.T) {
	t.Setenv("GOU_DEMO_STREAMING_TOOL_EXECUTION", "")
	t.Setenv("GOU_QUERY_STREAMING_PARITY", "1")
	tools, _ := json.Marshal([]map[string]any{{"name": "echo_stub", "input_schema": map[string]any{"type": "object"}}})

	deps := ProductionDeps()
	deps.StreamPost = func(ctx context.Context, p anthropicmessages.PostStreamParams) error {
		return anthropicmessages.ReadSSE(strings.NewReader(textOnlySSE()), func(data []byte) error {
			return anthropicmessages.ProcessStreamPayloads(data, p.Emit)
		})
	}

	ctx := context.Background()
	var got []types.MessageType
	for y, err := range Query(ctx, QueryParams{
		Messages: []types.Message{{
			Type: types.MessageTypeUser,
			UUID: "u1",
			Message: mustJSON(t, map[string]any{
				"role": "user", "content": "hi",
			}),
		}},
		SystemPrompt:    AsSystemPrompt([]string{"sys"}),
		ToolUseContext:  types.ToolUseContext{Options: types.ToolUseContextOptionsData{Tools: tools, MainLoopModel: "claude-3-5-haiku-20241022"}},
		StreamingParity: true,
		Deps:            &deps,
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if y.Message != nil {
			got = append(got, y.Message.Type)
		}
		if y.Terminal != nil {
			break
		}
	}
	if len(got) < 1 || got[0] != types.MessageTypeAssistant {
		t.Fatalf("got %#v", got)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sseToolUseRound is one assistant message with a single tool_use (stop_reason tool_use); see query.ts + Anthropic streaming.
func sseToolUseRound() string {
	return "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_t\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":2,\"output_tokens\":0}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_test\",\"name\":\"echo_stub\",\"input\":{}}}\n\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\",\"usage\":{\"output_tokens\":3}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
}

func TestStreamingParity_toolThenFollowUpText(t *testing.T) {
	t.Setenv("GOU_DEMO_STREAMING_TOOL_EXECUTION", "1")
	tools, _ := json.Marshal([]map[string]any{{
		"name":         "echo_stub",
		"description":  "echo",
		"input_schema": map[string]any{"type": "object", "properties": map[string]any{"message": map[string]any{"type": "string"}}},
	}})

	var calls int
	deps := ProductionDeps()
	deps.StreamPost = func(ctx context.Context, p anthropicmessages.PostStreamParams) error {
		calls++
		var body string
		if calls == 1 {
			body = sseToolUseRound()
		} else {
			body = textOnlySSE()
		}
		return anthropicmessages.ReadSSE(strings.NewReader(body), func(data []byte) error {
			return anthropicmessages.ProcessStreamPayloads(data, p.Emit)
		})
	}
	deps.ToolexecutionDeps = toolexecution.ExecutionDeps{
		RandomUUID: func() string { return "tool-result-uuid" },
		InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
			if name == "echo_stub" {
				return `{"echoed":"ok"}`, false, nil
			}
			return "", false, nil
		},
	}

	var got []types.MessageType
	ctx := context.Background()
	for y, err := range Query(ctx, QueryParams{
		Messages: []types.Message{{
			Type:    types.MessageTypeUser,
			UUID:    "u1",
			Message: mustJSON(t, map[string]any{"role": "user", "content": "run tool"}),
		}},
		SystemPrompt: AsSystemPrompt([]string{"sys"}),
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				Tools:         tools,
				MainLoopModel: "claude-3-5-haiku-20241022",
			},
		},
		StreamingParity: true,
		Deps:            &deps,
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if y.Message != nil {
			got = append(got, y.Message.Type)
		}
		if y.Terminal != nil {
			break
		}
	}
	if calls != 2 {
		t.Fatalf("want 2 API rounds, got %d", calls)
	}
	// assistant → user tool_result → assistant
	if len(got) < 3 {
		t.Fatalf("got len %d types %#v", len(got), got)
	}
	if got[0] != types.MessageTypeAssistant || got[1] != types.MessageTypeUser || got[2] != types.MessageTypeAssistant {
		t.Fatalf("sequence got %#v", got)
	}
}
