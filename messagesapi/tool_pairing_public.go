package messagesapi

import (
	"goc/gou/messagerow"
	"goc/types"
)

// ToolUseIDsInAssistant returns tool_use block ids in an assistant message (normalized).
func ToolUseIDsInAssistant(m types.Message) []string {
	m2 := messagerow.NormalizeMessageJSON(m)
	if m2.Type != types.MessageTypeAssistant {
		return nil
	}
	if err := ensureInnerFromContent(&m2); err != nil {
		return nil
	}
	return toolUseIDsInAssistant(&m2)
}

// AssistantHasToolUse reports whether the assistant message includes any tool_use block.
func AssistantHasToolUse(m types.Message) bool {
	return len(ToolUseIDsInAssistant(m)) > 0
}

// UserMessagesCoverAssistantToolUses returns true when every tool_use id from the assistant
// appears as a tool_result (tool_use_id) in the given user messages (any order across messages).
func UserMessagesCoverAssistantToolUses(asst types.Message, userMsgs []types.Message) bool {
	need := ToolUseIDsInAssistant(asst)
	if len(need) == 0 {
		return true
	}
	// Do not use IsEffectiveUserMessage alone: lenient / deep-only rows in userRun
	// must still contribute tool_result ids.
	have := make(map[string]struct{})
	for i := range userMsgs {
		u := &userMsgs[i]
		if IsExplicitAssistantMessage(*u) {
			continue
		}
		for _, id := range toolResultIDsInUserMessage(u) {
			have[id] = struct{}{}
		}
		for _, id := range toolResultUseIDsFromMessageDeep(u) {
			have[id] = struct{}{}
		}
	}
	for _, id := range need {
		if _, ok := have[id]; !ok {
			return false
		}
	}
	return true
}

func toolResultIDsInUserMessage(m *types.Message) []string {
	m2 := messagerow.NormalizeMessageJSON(*m)
	if !IsEffectiveUserMessage(m2) {
		return nil
	}
	if err := ensureInnerFromContent(&m2); err != nil {
		return nil
	}
	inner, err := getInner(&m2)
	if err != nil {
		return nil
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if id := toolResultBlockID(b); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// toolResultBlockID returns the tool use/call id for a tool_result content block
// (Anthropic tool_use_id or OpenAI-style tool_call_id on replay).
func toolResultBlockID(b map[string]any) string {
	if t, _ := b["type"].(string); t != "tool_result" {
		return ""
	}
	if id, _ := b["tool_use_id"].(string); id != "" {
		return id
	}
	if id, _ := b["tool_call_id"].(string); id != "" {
		return id
	}
	return ""
}
