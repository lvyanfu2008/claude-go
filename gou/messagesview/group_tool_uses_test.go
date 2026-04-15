package messagesview

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func ptr[T any](v T) *T { return &v }

func TestApplyGrouping(t *testing.T) {
	// Helper to create a tool use message
	toolUseMsg := func(id, uuid, messageID, toolName string) types.Message {
		content := []map[string]interface{}{
			{
				"type": "tool_use",
				"id":   id,
				"name": toolName,
			},
		}
		raw, _ := json.Marshal(content)
		return types.Message{
			Type:      types.MessageTypeAssistant,
			UUID:      uuid,
			MessageID: &messageID,
			Content:   raw,
		}
	}

	// Helper to create a tool result message
	toolResultMsg := func(id, uuid, toolUseID string) types.Message {
		content := []map[string]interface{}{
			{
				"type":        "tool_result",
				"tool_use_id": toolUseID,
			},
		}
		raw, _ := json.Marshal(content)
		return types.Message{
			Type:    types.MessageTypeUser,
			UUID:    uuid,
			Content: raw,
		}
	}

	tests := []struct {
		name    string
		verbose bool
		input   []types.Message
		wantLen int
		verify  func(*testing.T, []types.Message)
	}{
		{
			name:    "empty",
			verbose: false,
			input:   []types.Message{},
			wantLen: 0,
		},
		{
			name:    "verbose skips grouping",
			verbose: true,
			input: []types.Message{
				toolUseMsg("tu1", "1", "msg1", "Agent"),
				toolUseMsg("tu2", "2", "msg1", "Agent"),
			},
			wantLen: 2,
		},
		{
			name:    "groups 2+ agents",
			verbose: false,
			input: []types.Message{
				toolUseMsg("tu1", "1", "msg1", "Agent"),
				toolUseMsg("tu2", "2", "msg1", "Agent"),
			},
			wantLen: 1,
			verify: func(t *testing.T, out []types.Message) {
				if out[0].Type != types.MessageTypeGroupedToolUse {
					t.Errorf("Expected grouped_tool_use, got %s", out[0].Type)
				}
				if len(out[0].Messages) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(out[0].Messages))
				}
			},
		},
		{
			name:    "does not group single agent",
			verbose: false,
			input: []types.Message{
				toolUseMsg("tu1", "1", "msg1", "Agent"),
			},
			wantLen: 1,
			verify: func(t *testing.T, out []types.Message) {
				if out[0].Type != types.MessageTypeAssistant {
					t.Errorf("Expected assistant, got %s", out[0].Type)
				}
			},
		},
		{
			name:    "does not group non-Agent tools",
			verbose: false,
			input: []types.Message{
				toolUseMsg("tu1", "1", "msg1", "Bash"),
				toolUseMsg("tu2", "2", "msg1", "Bash"),
			},
			wantLen: 2,
		},
		{
			name:    "groups agents and removes their results",
			verbose: false,
			input: []types.Message{
				toolUseMsg("tu1", "1", "msg1", "Agent"),
				toolUseMsg("tu2", "2", "msg1", "Agent"),
				toolResultMsg("tr1", "3", "tu1"),
				toolResultMsg("tr2", "4", "tu2"),
				{Type: types.MessageTypeUser, UUID: "5", Content: []byte(`[{"type":"text","text":"hello"}]`)},
			},
			wantLen: 2, // 1 grouped_tool_use + 1 text message
			verify: func(t *testing.T, out []types.Message) {
				if out[0].Type != types.MessageTypeGroupedToolUse {
					t.Errorf("Expected grouped_tool_use, got %s", out[0].Type)
				}
				if len(out[0].Results) != 2 {
					t.Errorf("Expected 2 results, got %d", len(out[0].Results))
				}
				if out[1].UUID != "5" {
					t.Errorf("Expected text message, got %s", out[1].UUID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyGrouping(tt.input, tt.verbose)
			if len(got) != tt.wantLen {
				t.Errorf("ApplyGrouping() returned %d messages, want %d", len(got), tt.wantLen)
			}
			if tt.verify != nil {
				tt.verify(t, got)
			}
		})
	}
}
