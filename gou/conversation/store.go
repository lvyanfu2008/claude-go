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
