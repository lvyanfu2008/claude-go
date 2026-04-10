package messagerow

import (
	"encoding/json"

	"goc/types"
)

// NormalizeMessageJSON fills Content from embedded Message.{role,content} when Content is empty
// (API-shaped rows from transcript import or ccb-engine hydrate).
func NormalizeMessageJSON(msg types.Message) types.Message {
	if len(msg.Content) > 0 || len(msg.Message) == 0 {
		return msg
	}
	var inner struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &inner); err != nil || len(inner.Content) == 0 {
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
