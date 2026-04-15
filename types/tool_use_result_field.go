package types

import (
	"encoding/json"
	"strings"
)

// ToolUseResultJSONBytes prepares the JSON value for [Message.ToolUseResult].
//
// TS createUserMessage stores structured tool output as a parsed object on the user message.
// If we json.Marshal a Go string that already holds JSON, the field becomes a quoted string
// (double-encoded). When s is valid JSON whose top-level value is an object or array, we
// embed it raw; otherwise we JSON-encode s as a string (plain error text).
func ToolUseResultJSONBytes(s string) json.RawMessage {
	s = strings.TrimSpace(s)
	if s == "" {
		return json.RawMessage(`""`)
	}
	b := []byte(s)
	if json.Valid(b) {
		if len(b) > 0 {
			switch b[0] {
			case '{', '[':
				return json.RawMessage(b)
			case '"':
				// JSON string literal — embed as-is (valid RawMessage)
				return json.RawMessage(b)
			}
		}
	}
	out, err := json.Marshal(s)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(out)
}
