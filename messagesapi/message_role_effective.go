package messagesapi

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// IsExplicitAssistantMessage returns true for an assistant turn from top-level or explicit type.
func IsExplicitAssistantMessage(m types.Message) bool {
	ts := strings.TrimSpace(string(m.Type))
	if strings.EqualFold(ts, "assistant") {
		return true
	}
	if m.Type == types.MessageTypeAssistant {
		return true
	}
	if strings.ToLower(nestedRoleFromMessageField(&m)) == "assistant" {
		return true
	}
	return false
}

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

// IsEffectiveUserMessage returns true for a user turn. Transcript / hydrate can omit
// `type` and `message` while still storing tool_result only in `content` (the next row
// after an assistant is still a user turn for the API).
func IsEffectiveUserMessage(m types.Message) bool {
	ts := strings.TrimSpace(string(m.Type))
	if strings.EqualFold(ts, "user") {
		return true
	}
	if strings.EqualFold(ts, "assistant") {
		return false
	}
	switch m.Type {
	case types.MessageTypeSystem, types.MessageTypeProgress, types.MessageTypeAttachment,
		types.MessageTypeGroupedToolUse, types.MessageTypeCollapsedReadSearch:
		return false
	}
	r := strings.ToLower(nestedRoleFromMessageField(&m))
	if r == "user" {
		return true
	}
	if r == "assistant" {
		return false
	}
	// `type` empty and no embedded role: some JSONL has only `content: [{type:tool_result,...}]`
	if ts == "" && contentArrayHasToolResult(&m) {
		return true
	}
	return false
}

func contentArrayHasToolResult(m *types.Message) bool {
	if m == nil || len(m.Content) == 0 {
		return false
	}
	blocks, err := parseContentArrayOrString(m.Content)
	if err != nil {
		return false
	}
	for _, b := range blocks {
		if t, _ := b["type"].(string); t == "tool_result" {
			return true
		}
	}
	return false
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
