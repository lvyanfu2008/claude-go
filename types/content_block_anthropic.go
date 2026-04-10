// Message content blocks (assistant/user) — mirrors Anthropic API + src/components/Message.tsx branches.
// TS unions are flattened into one struct; unknown keys remain decodable via sibling RawMessage patterns where needed.
package types

import "encoding/json"

// MessageContentBlock mirrors common SDK / TS ContentBlockLike fields for JSON round-trip.
type MessageContentBlock struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   *bool           `json:"is_error,omitempty"`

	Thinking string `json:"thinking,omitempty"`

	// Server / advisor blocks (names align with Message.tsx cases).
	// Remaining fields on wire can be added here for full parity without breaking decode.
}
