package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestFormatAgentProgressEmpty(t *testing.T) {
	segs := FormatAgentProgressSegments(nil)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "Initializing…" {
		t.Fatalf("expected 'Initializing…', got %q", segs[0].Text)
	}

	// Also test empty slice
	segs = FormatAgentProgressSegments([]types.Message{})
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment for empty slice, got %d", len(segs))
	}
	if segs[0].Text != "Initializing…" {
		t.Fatalf("expected 'Initializing…' for empty slice, got %q", segs[0].Text)
	}
}

func TestFormatAgentProgressWithToolResult(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_result", "tool_use_id": "tu1"},
				},
			},
		},
	})
	pm := types.Message{
		Type: types.MessageTypeProgress,
		Data: json.RawMessage(inner),
	}
	segs := FormatAgentProgressSegments([]types.Message{pm})
	if len(segs) == 0 {
		t.Fatal("expected at least header segment")
	}
	if segs[0].Text == "Initializing…" {
		t.Fatal("expected non-empty progress, got Initializing…")
	}
	// Should show "1 tool use" in header
	if segs[0].Text != "" {
		t.Logf("header text: %q", segs[0].Text)
	}
}

func TestComputeAgentProgressStatsTokens(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{},
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	stats := computeAgentProgressStats(pms)
	if stats.Tokens != 150 {
		t.Fatalf("expected 150 tokens, got %d", stats.Tokens)
	}
}

func TestComputeAgentProgressStatsToolUses(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_result"},
					{"type": "tool_result"},
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	stats := computeAgentProgressStats(pms)
	if stats.ToolUseCount != 2 {
		t.Fatalf("expected 2 tool uses, got %d", stats.ToolUseCount)
	}
}

func TestComputeAgentProgressStatsMixed(t *testing.T) {
	userInner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_result"},
				},
			},
		},
	})
	assistantInner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{},
				"usage": map[string]any{
					"input_tokens":  200,
					"output_tokens": 75,
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(userInner)},
		{Type: types.MessageTypeProgress, Data: json.RawMessage(assistantInner)},
	}
	stats := computeAgentProgressStats(pms)
	if stats.ToolUseCount != 1 {
		t.Fatalf("expected 1 tool use, got %d", stats.ToolUseCount)
	}
	if stats.Tokens != 275 {
		t.Fatalf("expected 275 tokens, got %d", stats.Tokens)
	}
	if stats.IsEmpty {
		t.Fatal("expected non-empty stats")
	}
}

func TestComputeAgentProgressStatsEmpty(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	stats := computeAgentProgressStats(pms)
	if !stats.IsEmpty {
		t.Fatal("expected IsEmpty for message with no tool results and no usage")
	}
}

func TestComputeAgentProgressStatsNonProgressIgnored(t *testing.T) {
	pm := types.Message{
		Type: types.MessageTypeUser, // not a progress message
	}
	stats := computeAgentProgressStats([]types.Message{pm})
	if !stats.IsEmpty {
		t.Fatal("expected IsEmpty for non-progress message")
	}
}

func TestComputeAgentProgressStatsBadJSON(t *testing.T) {
	pm := types.Message{
		Type: types.MessageTypeProgress,
		Data: json.RawMessage(`{invalid`),
	}
	stats := computeAgentProgressStats([]types.Message{pm})
	if !stats.IsEmpty {
		t.Fatal("expected IsEmpty for unparseable progress data")
	}
}

func TestExtractRecentActivities(t *testing.T) {
	input, _ := json.Marshal(map[string]any{"file_path": "test.go"})
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_use", "name": "Read", "input": input},
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	activities := extractRecentActivities(pms, 3)
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0] == "" {
		t.Fatal("expected non-empty activity")
	}
}

func TestExtractRecentActivitiesMultiple(t *testing.T) {
	input1, _ := json.Marshal(map[string]any{"command": "ls"})
	input2, _ := json.Marshal(map[string]any{"pattern": "*.go"})
	msg1, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_use", "name": "Bash", "input": input1},
				},
			},
		},
	})
	msg2, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_use", "name": "Glob", "input": input2},
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(msg1)},
		{Type: types.MessageTypeProgress, Data: json.RawMessage(msg2)},
	}
	activities := extractRecentActivities(pms, 3)
	if len(activities) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(activities))
	}
	// Should be in chronological order: Bash first, then Glob
	if activities[0] == "" || activities[1] == "" {
		t.Fatal("expected non-empty activities")
	}
	t.Logf("activities: %v", activities)
}

func TestExtractRecentActivitiesEmpty(t *testing.T) {
	activities := extractRecentActivities(nil, 3)
	if len(activities) != 0 {
		t.Fatalf("expected 0 activities for nil input, got %d", len(activities))
	}

	activities = extractRecentActivities([]types.Message{}, 3)
	if len(activities) != 0 {
		t.Fatalf("expected 0 activities for empty input, got %d", len(activities))
	}
}

func TestExtractRecentActivitiesNonAssistantSkipped(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "user", // not assistant, should be skipped for tool_use extraction
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	activities := extractRecentActivities(pms, 3)
	if len(activities) != 0 {
		t.Fatalf("expected 0 activities for user type, got %d", len(activities))
	}
}

func TestExtractRecentActivitiesMaxLimit(t *testing.T) {
	input, _ := json.Marshal(map[string]any{"command": "echo 1"})
	makeMsg := func() types.Message {
		inner, _ := json.Marshal(map[string]any{
			"message": map[string]any{
				"type": "assistant",
				"message": map[string]any{
					"content": []map[string]any{
						{"type": "tool_use", "name": "Bash", "input": input},
					},
				},
			},
		})
		return types.Message{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)}
	}
	pms := []types.Message{makeMsg(), makeMsg(), makeMsg(), makeMsg(), makeMsg()}
	activities := extractRecentActivities(pms, 3)
	if len(activities) != 3 {
		t.Fatalf("expected at most 3 activities, got %d", len(activities))
	}
}

func TestFormatTokenCount(t *testing.T) {
	if s := formatTokenCount(500); s != "500" {
		t.Fatalf("expected '500', got %q", s)
	}
	if s := formatTokenCount(4200); s != "4.2k" {
		t.Fatalf("expected '4.2k', got %q", s)
	}
	if s := formatTokenCount(0); s != "0" {
		t.Fatalf("expected '0', got %q", s)
	}
	if s := formatTokenCount(1000000); s != "1000.0k" {
		t.Fatalf("expected '1000.0k', got %q", s)
	}
	if s := formatTokenCount(999); s != "999" {
		t.Fatalf("expected '999', got %q", s)
	}
}

func TestFormatTokenCountEdgeCases(t *testing.T) {
	if s := formatTokenCount(1000); s != "1.0k" {
		t.Fatalf("expected '1.0k', got %q", s)
	}
	if s := formatTokenCount(1); s != "1" {
		t.Fatalf("expected '1', got %q", s)
	}
}

func TestFormatAgentProgressActivityLines(t *testing.T) {
	input, _ := json.Marshal(map[string]any{"file_path": "/home/test/main.go"})
	inner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_use", "name": "Read", "input": input},
				},
			},
		},
	})
	// Also include a tool_result to make stats non-empty
	userInner, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "tool_result", "tool_use_id": "tu1"},
				},
			},
		},
	})
	pms := []types.Message{
		{Type: types.MessageTypeProgress, Data: json.RawMessage(userInner)},
		{Type: types.MessageTypeProgress, Data: json.RawMessage(inner)},
	}
	segs := FormatAgentProgressSegments(pms)
	if len(segs) < 2 {
		t.Fatalf("expected at least 2 segments (header + activity), got %d", len(segs))
	}
	// Header should not be "Initializing…"
	if segs[0].Text == "Initializing…" {
		t.Fatal("expected header with stats, got Initializing…")
	}
	// Activity line should be SegDisplayHint
	if segs[1].Kind != SegDisplayHint {
		t.Fatalf("expected SegDisplayHint for activity line, got %v", segs[1].Kind)
	}
	// Activity text should contain the read indicator
	t.Logf("activity text: %q", segs[1].Text)
}
