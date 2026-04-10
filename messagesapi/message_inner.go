package messagesapi

import (
	"encoding/json"

	"goc/types"
)

// ensureInnerFromContent builds nested `message` from top-level Content when Message is empty
// (rows that only have role implied by Type + Content).
func ensureInnerFromContent(m *types.Message) error {
	if len(m.Message) > 0 || len(m.Content) == 0 {
		return nil
	}
	role := "user"
	if m.Type == types.MessageTypeAssistant {
		role = "assistant"
	}
	inner := userOrAssistantInner{Role: role, Content: m.Content}
	return setInner(m, inner)
}

type userOrAssistantInner struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	ID      string          `json:"id,omitempty"`
	Model   string          `json:"model,omitempty"`
}

func getInner(m *types.Message) (userOrAssistantInner, error) {
	var inner userOrAssistantInner
	if len(m.Message) == 0 {
		return inner, nil
	}
	if err := json.Unmarshal(m.Message, &inner); err != nil {
		return inner, err
	}
	return inner, nil
}

func setInner(m *types.Message, inner userOrAssistantInner) error {
	b, err := json.Marshal(inner)
	if err != nil {
		return err
	}
	m.Message = b
	return nil
}

// parseContentArrayOrString returns content blocks; string content becomes one text block.
func parseContentArrayOrString(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil, nil
		}
		return []map[string]any{{"type": "text", "text": s}}, nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func marshalContentBlocks(blocks []map[string]any) (json.RawMessage, error) {
	if blocks == nil {
		return json.RawMessage("[]"), nil
	}
	return json.Marshal(blocks)
}

func cloneMessage(m types.Message) (types.Message, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return m, err
	}
	var out types.Message
	err = json.Unmarshal(b, &out)
	return out, err
}

func isTruthy(p *bool) bool {
	return p != nil && *p
}

func lastIdx[T any](s []T) int {
	return len(s) - 1
}
