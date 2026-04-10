// Mirrors @anthropic-ai/sdk ContentBlockParam used in processUserInput (text | image, …).
package types

import "encoding/json"

// ContentBlockParam is a minimal JSON shape for user message blocks.
// Image and other variants often nest `source`; kept as RawMessage for parity with TS unions.
type ContentBlockParam struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	Source json.RawMessage `json:"source,omitempty"`
}
