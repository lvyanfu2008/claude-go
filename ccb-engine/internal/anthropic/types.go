package anthropic

import "encoding/json"

// Message is one API message (role + content).
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentBlock
}

// ContentBlock is assistant or user structured content.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	ToolUseID string `json:"tool_use_id,omitempty"`
	// Content is tool_result body (string or blocks)
	Content any `json:"content,omitempty"`
	IsError bool `json:"is_error,omitempty"`
}
