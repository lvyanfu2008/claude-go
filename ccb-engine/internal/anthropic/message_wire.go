package anthropic

import "encoding/json"

// CanonicalizeMessages normalizes each message's Content after json.Unmarshal into []Message.
// Unmarshaling JSON arrays into `any` yields []interface{} holding map[string]interface{} per element;
// re-binding to []ContentBlock makes encoding/json emit a stable []object wire shape (same as TS SDK
// when message.message.content is a ContentBlockParam[]), instead of relying on map-in-slice any encoding.
func CanonicalizeMessages(msgs []Message) []Message {
	out := make([]Message, len(msgs))
	for i := range msgs {
		out[i] = canonicalizeOneMessage(msgs[i])
	}
	return out
}

func canonicalizeOneMessage(m Message) Message {
	switch m.Content.(type) {
	case nil, string, []ContentBlock:
		return m
	default:
		raw, err := json.Marshal(m.Content)
		if err != nil {
			return m
		}
		var blocks []ContentBlock
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return m
		}
		return Message{
			Role:            m.Role,
			Content:         blocks,
			Type:            m.Type,
			Subtype:         m.Subtype,
			CompactMetadata: m.CompactMetadata,
		}
	}
}
