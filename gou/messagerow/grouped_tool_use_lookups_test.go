package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func ptr[T any](v T) *T { return &v }

func TestBuildGroupedAgentLookups(t *testing.T) {
	// Helper to create a progress message
	progressMsg := func(parentID string, data string) types.Message {
		raw, _ := json.Marshal(map[string]interface{}{
			"parentToolUseID": parentID,
			"other":           data,
		})
		return types.Message{
			Type:            types.MessageTypeProgress,
			ParentToolUseID: ptr(parentID),
			Data:            raw,
		}
	}

	// Helper to create a tool use message
	toolUseMsg := func(id, name string) types.Message {
		content := []map[string]interface{}{
			{
				"type": "tool_use",
				"id":   id,
				"name": name,
			},
		}
		raw, _ := json.Marshal(content)
		return types.Message{
			Type:    types.MessageTypeAssistant,
			Content: raw,
		}
	}

	// Helper to create a tool result message
	toolResultMsg := func(id string, isError bool) types.Message {
		content := []map[string]interface{}{
			{
				"type":        "tool_result",
				"tool_use_id": id,
				"is_error":    isError,
			},
		}
		raw, _ := json.Marshal(content)
		return types.Message{
			Type:    types.MessageTypeUser,
			Content: raw,
		}
	}

	messages := []types.Message{
		toolUseMsg("tu1", "Agent"),
		toolUseMsg("tu2", "Agent"),
		toolResultMsg("tu1", false),
		toolResultMsg("tu2", true),
		toolUseMsg("tu3", "Agent"),
		progressMsg("tu3", "data1"),
		progressMsg("tu3", "data2"),
	}

	lookups := BuildGroupedAgentLookups(messages)

	// Test resolved/errored
	if !lookups.ResolvedToolUseIDs["tu1"] {
		t.Errorf("tu1 should be resolved")
	}
	if lookups.ErroredToolUseIDs["tu1"] {
		t.Errorf("tu1 should not be errored")
	}

	if !lookups.ResolvedToolUseIDs["tu2"] {
		t.Errorf("tu2 should be resolved")
	}
	if !lookups.ErroredToolUseIDs["tu2"] {
		t.Errorf("tu2 should be errored")
	}

	if lookups.ResolvedToolUseIDs["tu3"] {
		t.Errorf("tu3 should not be resolved")
	}

	// Test in-progress
	if lookups.InProgressToolUseIDs["tu1"] || lookups.InProgressToolUseIDs["tu2"] {
		t.Errorf("tu1/tu2 should not be in-progress")
	}
	if !lookups.InProgressToolUseIDs["tu3"] {
		t.Errorf("tu3 should be in-progress")
	}

	// Test progress mapping
	if len(lookups.ProgressMessagesByToolUseID["tu3"]) != 2 {
		t.Errorf("tu3 should have 2 progress messages, got %d", len(lookups.ProgressMessagesByToolUseID["tu3"]))
	}
}
