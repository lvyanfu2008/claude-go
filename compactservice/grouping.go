package compactservice

import (
	"encoding/json"

	"goc/types"
)

// GroupMessagesByApiRound mirrors groupMessagesByApiRound in
// claude-code/src/services/compact/grouping.ts. Splits the message stream at the
// start of each new assistant turn (detected by assistant message.id changing).
// Preserves TS semantics for resumed/malformed conversations: a missing or
// dangling assistant id falls through the boundary gate.
func GroupMessagesByApiRound(messages []types.Message) [][]types.Message {
	var groups [][]types.Message
	var current []types.Message
	lastAssistantID := ""

	for _, msg := range messages {
		isAssistant := msg.Type == types.MessageTypeAssistant
		currID := ""
		if isAssistant {
			currID = assistantInnerMessageID(msg)
		}
		if isAssistant && currID != lastAssistantID && len(current) > 0 {
			groups = append(groups, current)
			current = []types.Message{msg}
		} else {
			current = append(current, msg)
		}
		if isAssistant {
			lastAssistantID = currID
		}
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

// assistantInnerMessageID extracts the inner message.id field on an assistant message.
// This is the canonical boundary key — streaming chunks share an id, new API rounds don't.
func assistantInnerMessageID(m types.Message) string {
	if m.MessageID != nil {
		return *m.MessageID
	}
	if len(m.Message) == 0 {
		return ""
	}
	var env struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(m.Message, &env); err != nil {
		return ""
	}
	return env.ID
}
