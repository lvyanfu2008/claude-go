package messagerow

import (
	"bytes"
	"encoding/json"

	"goc/types"
)

// contentIsEmptyForMerge reports whether Content should be treated as unset so
// Message.{role,content} can be merged. Persisted rows sometimes use "content": []
// which is non-empty []byte but must not block merging the nested message field.
func contentIsEmptyForMerge(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	s := bytes.TrimSpace(raw)
	if len(s) == 0 {
		return true
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(s, &arr); err != nil {
		return false
	}
	return len(arr) == 0
}

// NormalizeMessageJSON fills Content from embedded Message.{role,content} when Content is empty
// (API-shaped rows from transcript import or ccb-engine hydrate).
func NormalizeMessageJSON(msg types.Message) types.Message {
	if !contentIsEmptyForMerge(msg.Content) || len(msg.Message) == 0 {
		return msg
	}
	var inner struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &inner); err != nil || contentIsEmptyForMerge(inner.Content) {
		return msg
	}
	var asString string
	if err := json.Unmarshal(inner.Content, &asString); err == nil {
		raw, err := json.Marshal([]map[string]string{{"type": "text", "text": asString}})
		if err != nil {
			return msg
		}
		msg.Content = raw
		return msg
	}
	msg.Content = inner.Content
	return msg
}
