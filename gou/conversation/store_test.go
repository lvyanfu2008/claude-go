package conversation

import (
	"testing"

	"goc/types"
)

func TestItemKey(t *testing.T) {
	m := types.Message{UUID: "abc", Type: types.MessageTypeUser}
	if got := ItemKey(m, "sess"); got != "abc-sess" {
		t.Fatalf("got %q", got)
	}
}

func TestStore_ItemKeys(t *testing.T) {
	s := Store{
		ConversationID: "c1",
		Messages: []types.Message{
			{UUID: "u1", Type: types.MessageTypeUser},
			{UUID: "a1", Type: types.MessageTypeAssistant},
		},
	}
	keys := s.ItemKeys()
	if len(keys) != 2 || keys[0] != "u1-c1" || keys[1] != "a1-c1" {
		t.Fatalf("%v", keys)
	}
}

func TestStore_streaming(t *testing.T) {
	var s Store
	s.AppendStreamingChunk("hel")
	s.AppendStreamingChunk("lo")
	if s.StreamingText != "hello" {
		t.Fatal(s.StreamingText)
	}
	s.ClearStreaming()
	if s.StreamingText != "" {
		t.Fatal()
	}
}
