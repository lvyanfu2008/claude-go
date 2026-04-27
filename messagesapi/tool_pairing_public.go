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
	return toolUseIDsInAssistantBlocks(&m2)
}

func toolUseIDsInAssistantBlocks(m *types.Message) []string {
	inner, err := getInner(m)
	if err != nil {
		return nil
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if t, _ := b["type"].(string); t != "tool_use" {
			continue
		}
		id, _ := b["id"].(string)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// AssistantHasToolUse reports whether the assistant message includes any tool_use block.
func AssistantHasToolUse(m types.Message) bool {
	return len(ToolUseIDsInAssistant(m)) > 0
}

// UserMessagesCoverAssistantToolUses returns true when every tool_use id from the assistant
// appears as a tool_result in the given user messages (Type user; tool_use_id / tool_call_id in content).
func UserMessagesCoverAssistantToolUses(asst types.Message, userMsgs []types.Message) bool {
	need := ToolUseIDsInAssistant(asst)
	if len(need) == 0 {
		return true
	}
	have := make(map[string]struct{})
	for i := range userMsgs {
		if userMsgs[i].Type != types.MessageTypeUser {
			continue
		}
		for _, id := range toolResultIDsInUserMessageStrict(&userMsgs[i]) {
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

func toolResultIDsInUserMessageStrict(m *types.Message) []string {
	m2 := messagerow.NormalizeMessageJSON(*m)
	if m2.Type != types.MessageTypeUser {
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
		if t, _ := b["type"].(string); t != "tool_result" {
			continue
		}
		if id, _ := b["tool_use_id"].(string); id != "" {
			ids = append(ids, id)
		}
		if id, _ := b["tool_call_id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
