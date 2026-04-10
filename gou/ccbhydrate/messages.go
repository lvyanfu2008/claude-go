// Package ccbhydrate builds JSON message arrays for ccb-engine SubmitUserTurn / HydrateFromMessages
// (goc/ccb-engine/internal/anthropic.Message: role + content string or block array).
package ccbhydrate

import (
	"encoding/json"
	"strings"

	"goc/gou/messagerow"
	"goc/messagesapi"
	"goc/types"
)

// apiMessage is the on-wire shape json.Unmarshal uses into engine session.
type apiMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// MessagesJSONNormalized runs [messagesapi.NormalizeMessagesForAPI] then the same JSON projection as [MessagesJSON].
func MessagesJSONNormalized(msgs []types.Message, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	norm, err := messagesapi.NormalizeMessagesForAPI(msgs, tools, opts)
	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(norm)
}

// MessagesJSON returns a JSON array suitable for SubmitUserTurn payload.messages.
// Only user and assistant rows with non-empty content (after messagerow.NormalizeMessageJSON) are included, in order.
func MessagesJSON(msgs []types.Message) (json.RawMessage, error) {
	norm := make([]types.Message, len(msgs))
	for i, m := range msgs {
		norm[i] = messagerow.NormalizeMessageJSON(m)
	}
	return messagesJSONFromNormalized(norm)
}

func messagesJSONFromNormalized(msgs []types.Message) (json.RawMessage, error) {
	var out []apiMessage
	for _, m := range msgs {
		m = messagerow.NormalizeMessageJSON(m)
		switch m.Type {
		case types.MessageTypeUser, types.MessageTypeAssistant:
		default:
			continue
		}
		if len(m.Content) == 0 {
			continue
		}
		role := string(m.Type)
		out = append(out, apiMessage{Role: role, Content: m.Content})
	}
	if len(out) == 0 {
		return json.RawMessage("[]"), nil
	}
	return json.Marshal(out)
}

// MessagesJSONWithSkillListing prepends one user message whose string content is listingUserText (already system-reminder wrapped if required),
// then the messages from [MessagesJSONNormalized] with the given tools and options. listingUserText empty returns the same as that base.
func MessagesJSONWithSkillListing(msgs []types.Message, listingUserText string, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	base, err := MessagesJSONNormalized(msgs, tools, opts)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(listingUserText) == "" {
		return base, nil
	}
	var arr []apiMessage
	if err := json.Unmarshal(base, &arr); err != nil {
		return nil, err
	}
	content, err := json.Marshal(listingUserText)
	if err != nil {
		return nil, err
	}
	prepended := append([]apiMessage{{Role: "user", Content: content}}, arr...)
	return json.Marshal(prepended)
}

// PrependUserMessageJSON prepends one user message with string content (JSON string) to a messages array.
// Empty trimmed text returns base unchanged.
func PrependUserMessageJSON(base json.RawMessage, text string) (json.RawMessage, error) {
	if strings.TrimSpace(text) == "" {
		return base, nil
	}
	var arr []apiMessage
	if err := json.Unmarshal(base, &arr); err != nil {
		return nil, err
	}
	content, err := json.Marshal(text)
	if err != nil {
		return nil, err
	}
	prepended := append([]apiMessage{{Role: "user", Content: content}}, arr...)
	return json.Marshal(prepended)
}
