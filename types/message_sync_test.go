package types

import (
	"encoding/json"
	"testing"
)

func TestSyncAssistantMessageID(t *testing.T) {
	inner, err := json.Marshal(map[string]any{
		"role":    "assistant",
		"id":      "msg_api_1",
		"content": []any{map[string]any{"type": "text", "text": "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := Message{Type: MessageTypeAssistant, Message: inner}
	SyncAssistantMessageID(&m)
	if m.MessageID == nil || *m.MessageID != "msg_api_1" {
		t.Fatalf("MessageID=%v", m.MessageID)
	}
}

func TestSyncAssistantMessageID_skipsWhenSet(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{"role": "assistant", "id": "inner"})
	other := "grouped-mid"
	m := Message{Type: MessageTypeAssistant, Message: inner, MessageID: &other}
	SyncAssistantMessageID(&m)
	if m.MessageID == nil || *m.MessageID != "grouped-mid" {
		t.Fatal("expected existing MessageID kept")
	}
}
