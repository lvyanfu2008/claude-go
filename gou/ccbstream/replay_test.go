package ccbstream

import (
	"strings"
	"testing"

	"goc/gou/conversation"
)

func TestReplayReader(t *testing.T) {
	r := strings.NewReader(`{"type":"assistant_delta","text":"x"}` + "\n" + `{"type":"turn_complete"}` + "\n")
	st := &conversation.Store{ConversationID: "r"}
	if err := ReplayReader(r, st); err != nil {
		t.Fatal(err)
	}
	if len(st.Messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(st.Messages))
	}
}
