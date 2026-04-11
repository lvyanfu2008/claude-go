package messagesapi

import (
	"encoding/json"
	"strings"
)

// UserAllTextContentAsJSONString re-encodes user message `content` as a JSON string when it is
// a JSON array of only `{ "type": "text", "text": "..." }` blocks (no tool_result / images / etc.).
// This matches Anthropic's `content: string | ContentBlock[]` and common Claude Code TS API dumps
// where all-text turns are sent as one string (concatenated block texts; same semantics as adjacent
// text blocks with no extra separator between blocks).
//
// Returns ok false if raw is empty, already a JSON string, not an array, has any non-text block,
// or marshalling fails.
func UserAllTextContentAsJSONString(raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 || raw[0] != '[' {
		return nil, false
	}
	blocks, err := parseContentArrayOrString(raw)
	if err != nil || len(blocks) == 0 {
		return nil, false
	}
	var sb strings.Builder
	for _, b := range blocks {
		t, _ := b["type"].(string)
		if t != "text" {
			return nil, false
		}
		tx, _ := b["text"].(string)
		sb.WriteString(tx)
	}
	out, err := json.Marshal(sb.String())
	if err != nil {
		return nil, false
	}
	return out, true
}
