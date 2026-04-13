package ccbstream

import (
	"strings"
	"testing"

	"goc/gou/conversation"
	"goc/types"
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

func TestApply_usageAccumulates(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{Type: "usage", InputTokens: 100, OutputTokens: 20})
	Apply(st, StreamEvent{Type: "usage", InputTokens: 30, OutputTokens: 5})
	if st.UsageInputTotal != 130 || st.UsageOutputTotal != 25 || st.TotalUsageTokens() != 155 {
		t.Fatalf("usage %+v", st)
	}
}

func TestApply_toolResultCollapsesReadSearchTail(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{Type: "tool_use", ID: "r1", Name: "Read", Input: map[string]any{"file_path": "/tmp/a.txt"}})
	Apply(st, StreamEvent{Type: "tool_result", ToolUseID: "r1", Content: "body"})
	if len(st.Messages) != 1 {
		t.Fatalf("messages %d", len(st.Messages))
	}
	if st.Messages[0].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("type=%s", st.Messages[0].Type)
	}
}

func TestApply_executeToolPlaceholder(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	Apply(st, StreamEvent{
		Type:      "execute_tool",
		Name:      "Bash",
		ToolUseID: "tu-1",
		CallID:    "c1",
	})
	if len(st.Messages) != 1 {
		t.Fatalf("messages %d", len(st.Messages))
	}
	raw := string(st.Messages[0].Content)
	if !strings.Contains(raw, "execute_tool") || !strings.Contains(raw, "Bash") || !strings.Contains(raw, "tu-1") {
		t.Fatalf("placeholder content: %s", raw)
	}
}
