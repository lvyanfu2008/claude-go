package ccbstream

import (
	"strings"
	"testing"

	"goc/gou/conversation"
)

func TestApply_deltaAndTurnComplete(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{Type: "assistant_delta", Text: "Hello "})
	Apply(st, StreamEvent{Type: "assistant_delta", Text: "world"})
	if strings.TrimSpace(st.StreamingText) != "Hello world" {
		t.Fatalf("streaming %q", st.StreamingText)
	}
	Apply(st, StreamEvent{Type: "turn_complete"})
	if st.StreamingText != "" {
		t.Fatalf("expected cleared stream, got %q", st.StreamingText)
	}
	if len(st.Messages) != 1 {
		t.Fatalf("messages %d", len(st.Messages))
	}
}

func TestApply_responseEndFlushesStreaming(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{Type: "assistant_delta", Text: "only response_end"})
	Apply(st, StreamEvent{Type: "response_end", ID: "r1"})
	if st.StreamingText != "" {
		t.Fatalf("expected stream cleared, got %q", st.StreamingText)
	}
	if len(st.Messages) != 1 {
		t.Fatalf("want 1 assistant message, got %d", len(st.Messages))
	}
}

func TestApply_toolUse(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{Type: "tool_use", ID: "x1", Name: "Bash", Input: map[string]any{"command": "ls"}})
	if len(st.Messages) != 1 {
		t.Fatalf("messages %d", len(st.Messages))
	}
}
