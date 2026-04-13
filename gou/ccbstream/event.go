// Package ccbstream applies ccb-engine NDJSON stream events to a conversation.Store.
// JSON shapes mirror goc/ccb-engine/internal/protocol (that package is internal to ccb-engine).
package ccbstream

// StreamEvent is server→client NDJSON line (protocol-v1.md StreamEvent union).
type StreamEvent struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolUseID  string         `json:"tool_use_id,omitempty"`
	CallID     string         `json:"call_id,omitempty"`
	Content    string         `json:"content,omitempty"`
	StateRev   uint64         `json:"state_rev,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Code       string         `json:"code,omitempty"`
	Message    string         `json:"message,omitempty"`

	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`

	IsError bool `json:"is_error,omitempty"`

	// Policy is optional on execute_tool (protocol-v1); ignored by Apply except for future use.
	Policy map[string]any `json:"policy,omitempty"`
}
