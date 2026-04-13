package conversation

// StreamingToolUse mirrors claude-code StreamingToolUse (messages.ts): content block index,
// stable tool_use id, name, and growing partial JSON during SSE input_json_delta events.
type StreamingToolUse struct {
	Index         int
	ToolUseID     string
	Name          string
	UnparsedInput string
}

// ClearStreamingToolUses resets the live streaming tool-use list (TS setStreamingToolUses([]) on message_stop / query start).
func (s *Store) ClearStreamingToolUses() {
	if s == nil {
		return
	}
	s.StreamingToolUses = nil
}

// AppendStreamingToolUse adds one in-flight row (TS content_block_start for tool_use). Used when wiring SSE parity or tests.
func (s *Store) AppendStreamingToolUse(u StreamingToolUse) {
	if s == nil {
		return
	}
	s.StreamingToolUses = append(s.StreamingToolUses, u)
}
