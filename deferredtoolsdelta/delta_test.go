package deferredtoolsdelta

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestGetDeferredToolsDelta_FirstCall(t *testing.T) {
	names := []string{"TaskCreate", "TaskGet", "CronCreate"}
	delta := GetDeferredToolsDelta(names, nil)
	if delta == nil {
		t.Fatal("expected delta on first call")
	}
	if len(delta.AddedNames) != 3 {
		t.Fatalf("expected 3 added, got %d", len(delta.AddedNames))
	}
}

func TestGetDeferredToolsDelta_NoChange(t *testing.T) {
	names := []string{"TaskCreate", "TaskGet"}
	// Simulate a previous announcement via user message
	content, _ := json.Marshal("<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>")
	msgBytes, _ := json.Marshal(map[string]any{"role": "user", "content": "<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>"})
	msgs := []types.Message{
		{
			Type:    types.MessageTypeUser,
			Message: json.RawMessage(msgBytes),
			Content: json.RawMessage(content),
		},
	}
	delta := GetDeferredToolsDelta(names, msgs)
	if delta != nil {
		t.Fatalf("expected nil delta, got added=%v", delta.AddedNames)
	}
}

func TestGetDeferredToolsDelta_NewToolAdded(t *testing.T) {
	names := []string{"TaskCreate", "TaskGet", "CronCreate"}
	content, _ := json.Marshal("<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>")
	msgBytes, _ := json.Marshal(map[string]any{"role": "user", "content": "<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>"})
	msgs := []types.Message{
		{
			Type:    types.MessageTypeUser,
			Message: json.RawMessage(msgBytes),
			Content: json.RawMessage(content),
		},
	}
	delta := GetDeferredToolsDelta(names, msgs)
	if delta == nil {
		t.Fatal("expected delta for new tool")
	}
	if len(delta.AddedNames) != 1 || delta.AddedNames[0] != "CronCreate" {
		t.Fatalf("expected [CronCreate] added, got %v", delta.AddedNames)
	}
}

func TestGetDeferredToolsDelta_ToolRemoved(t *testing.T) {
	names := []string{"TaskCreate"}
	content, _ := json.Marshal("<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>")
	msgBytes, _ := json.Marshal(map[string]any{"role": "user", "content": "<system-reminder>\n<available-deferred-tools>\nTaskCreate\nTaskGet\n</available-deferred-tools>\n</system-reminder>"})
	msgs := []types.Message{
		{
			Type:    types.MessageTypeUser,
			Message: json.RawMessage(msgBytes),
			Content: json.RawMessage(content),
		},
	}
	delta := GetDeferredToolsDelta(names, msgs)
	if delta == nil {
		t.Fatal("expected delta for removed tool")
	}
	if len(delta.RemovedNames) != 1 || delta.RemovedNames[0] != "TaskGet" {
		t.Fatalf("expected [TaskGet] removed, got %v", delta.RemovedNames)
	}
}

func TestBuildDeferredToolsSystemReminder(t *testing.T) {
	reminder := BuildDeferredToolsSystemReminder([]string{"TaskCreate", "CronCreate"})
	if reminder == "" {
		t.Fatal("expected non-empty reminder")
	}
	if !contains(reminder, "TaskCreate") || !contains(reminder, "CronCreate") {
		t.Fatal("missing tool names in reminder")
	}
	if !contains(reminder, "available-deferred-tools") {
		t.Fatal("missing <available-deferred-tools>")
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
