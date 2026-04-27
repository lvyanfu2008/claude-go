package messagesapi

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// nestedRoleFromMessageField returns message.role from the nested JSON when present
// (transcript lines sometimes have empty top-level type).
func nestedRoleFromMessageField(m *types.Message) string {
	if m == nil || len(m.Message) == 0 {
		return ""
	}
	var inner struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(m.Message, &inner) != nil {
		return ""
	}
	return strings.TrimSpace(inner.Role)
}

// IsEffectiveUserMessage returns true for a user turn: explicit Type, or when Type is
// empty/unknown and nested message.role is "user" (API-shaped or legacy JSONL).
func IsEffectiveUserMessage(m types.Message) bool {
	switch m.Type {
	case types.MessageTypeUser:
		return true
	case types.MessageTypeAssistant, types.MessageTypeSystem, types.MessageTypeProgress,
		types.MessageTypeAttachment, types.MessageTypeGroupedToolUse, types.MessageTypeCollapsedReadSearch:
		return false
	default:
		return nestedRoleFromMessageField(&m) == "user"
	}
}

// isInterstitialBeforeToolUserRow returns true for non-user rows that may appear
// between an assistant and the tool-result user in on-disk buffers (not the next turn).
func isInterstitialBeforeToolUserRow(m types.Message) bool {
	if IsEffectiveUserMessage(m) {
		return false
	}
	if m.Type == types.MessageTypeProgress {
		return true
	}
	if m.Type == types.MessageTypeSystem {
		return true
	}
	if m.Type == types.MessageTypeAttachment {
		return true
	}
	if m.Type == types.MessageTypeGroupedToolUse {
		return true
	}
	if m.Type == types.MessageTypeCollapsedReadSearch {
		return true
	}
	return false
}
