// Package conversation holds in-memory transcript state for a TUI (TS: Messages / AppState messages).
package conversation

import (
	"goc/types"
)

// Store is a minimal transcript + streaming buffer (streamingText lives outside renderable slice in TS).
type Store struct {
	ConversationID string
	Messages       []types.Message
	StreamingText  string

	// StreamingToolUses mirrors REPL.tsx streamingToolUses (in-flight tool_use rows before the turn finalizes).
	// NDJSON ccbstream typically keeps this empty (atomic tool_use lines); HTTP SSE wiring can append deltas later.
	// Cleared on turn_complete / response_end (TS message_stop), query turn boundaries, and gou-demo fake stream end.
	StreamingToolUses []StreamingToolUse

	// UsageInputTotal / UsageOutputTotal sum ccbstream "usage" events (TS getTotalInputTokens / getTotalOutputTokens path).
	UsageInputTotal  int
	UsageOutputTotal int
}

// ItemKey mirrors Messages.tsx messageKey: `${uuid}-${conversationId}`.
func ItemKey(m types.Message, conversationID string) string {
	return m.UUID + "-" + conversationID
}

// ItemKeys returns keys for virtual scroll (only persisted messages, not streaming tail).
func (s *Store) ItemKeys() []string {
	out := make([]string, len(s.Messages))
	for i := range s.Messages {
		out[i] = ItemKey(s.Messages[i], s.ConversationID)
	}
	return out
}

// AppendStreamingChunk appends raw assistant SSE text (TS: streamingText += chunk).
func (s *Store) AppendStreamingChunk(text string) {
	s.StreamingText += text
}

// ClearStreaming resets the live stream buffer (e.g. after turn finalized).
func (s *Store) ClearStreaming() {
	s.StreamingText = ""
}

// AppendMessage appends a normalized message (caller sets Type / UUID / fields).
func (s *Store) AppendMessage(m types.Message) {
	s.Messages = append(s.Messages, m)
}

// AddUsage accumulates token counts from stream usage lines (Anthropic per-turn usage on the wire).
func (s *Store) AddUsage(inputTokens, outputTokens int) {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	s.UsageInputTotal += inputTokens
	s.UsageOutputTotal += outputTokens
}

// TotalUsageTokens returns input+output totals for status UI.
func (s *Store) TotalUsageTokens() int {
	return s.UsageInputTotal + s.UsageOutputTotal
}
