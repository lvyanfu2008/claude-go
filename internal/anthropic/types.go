package anthropic

import (
	"bytes"
	"encoding/json"
)

// Message is one API message (role + content), optionally carrying transcript-only
// fields (type/subtype/compactMetadata) that MUST NOT be sent on the Messages API wire;
// MarshalJSON emits only role+content (mirrors TS normalize → API projection).
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentBlock
	// Transcript-only (e.g. compact_boundary); omitted on API wire via MarshalJSON.
	Type            string          `json:"type,omitempty"`
	Subtype         string          `json:"subtype,omitempty"`
	CompactMetadata json.RawMessage `json:"compactMetadata,omitempty"`
}

// MarshalJSON restricts the wire shape to role + content for POST /v1/messages.
func (m Message) MarshalJSON() ([]byte, error) {
	type wire struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(wire{Role: m.Role, Content: m.Content}); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
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
	Content any  `json:"content,omitempty"`
	IsError bool `json:"is_error,omitempty"`
}
