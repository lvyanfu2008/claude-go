package messagesapi

import (
	"encoding/json"

	"goc/gou/messagerow"
	"goc/types"
)

// RepairToolUseToolResultPairing ensures every assistant message that contains
// tool_use blocks is immediately followed by user message(s) whose content
// includes a tool_result for each tool_use id. Missing entries are appended
// as synthetic error tool_result blocks (or a new user row is inserted).
// This matches the API contract required by Anthropic and OpenAI tool-calling
// (each tool_call_id / tool_use id must have a following tool result).
func RepairToolUseToolResultPairing(messages []types.Message) ([]types.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}
	out := make([]types.Message, 0, len(messages)+2)
	for i := 0; i < len(messages); i++ {
		m := messagerow.NormalizeMessageJSON(messages[i])
		if m.Type != types.MessageTypeAssistant {
			out = append(out, m)
			continue
		}
		if err := ensureInnerFromContent(&m); err != nil {
			return nil, err
		}
		required := toolUseIDsInAssistant(&m)
		out = append(out, m)
		if len(required) == 0 {
			continue
		}
		asstUUID := m.UUID
		if i+1 < len(messages) {
			next := messagerow.NormalizeMessageJSON(messages[i+1])
			if next.Type == types.MessageTypeUser {
				patched, err := patchUserWithMissingToolResults(next, required, asstUUID)
				if err != nil {
					return nil, err
				}
				out = append(out, patched)
				i++
				continue
			}
		}
		out = append(out, syntheticUserWithToolResults(required, asstUUID))
	}
	return out, nil
}

func toolUseIDsInAssistant(m *types.Message) []string {
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

func patchUserWithMissingToolResults(u types.Message, required []string, sourceAsstUUID string) (types.Message, error) {
	m, err := cloneMessage(u)
	if err != nil {
		return types.Message{}, err
	}
	if err := ensureInnerFromContent(&m); err != nil {
		return types.Message{}, err
	}
	inner, err := getInner(&m)
	if err != nil {
		return types.Message{}, err
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return types.Message{}, err
	}
	have := make(map[string]struct{})
	for _, b := range blocks {
		if t, _ := b["type"].(string); t != "tool_result" {
			continue
		}
		tid, _ := b["tool_use_id"].(string)
		if tid != "" {
			have[tid] = struct{}{}
		}
	}
	for _, id := range required {
		if _, ok := have[id]; ok {
			continue
		}
		blocks = append(blocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": id,
			"is_error":    true,
			"content":     "<tool_use_error>Error: missing tool result (client repair)</tool_use_error>",
		})
	}
	blocks = hoistToolResults(blocks)
	raw, err := marshalContentBlocks(blocks)
	if err != nil {
		return types.Message{}, err
	}
	inner.Content = raw
	if err := setInner(&m, inner); err != nil {
		return types.Message{}, err
	}
	syncTopLevelContent(&m)
	if sourceAsstUUID != "" {
		m.SourceToolAssistantUUID = &sourceAsstUUID
	}
	return m, nil
}

func syntheticUserWithToolResults(ids []string, sourceAsstUUID string) types.Message {
	blocks := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		blocks = append(blocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": id,
			"is_error":    true,
			"content":     "<tool_use_error>Error: missing tool result row (client repair)</tool_use_error>",
		})
	}
	blocks = hoistToolResults(blocks)
	raw, err := marshalContentBlocks(blocks)
	if err != nil {
		raw = json.RawMessage(`[]`)
	}
	msg := createUserMessageFromContent(raw, randomUUID(), "", false)
	if sourceAsstUUID != "" {
		msg.SourceToolAssistantUUID = &sourceAsstUUID
	}
	tr, _ := json.Marshal("Error: missing tool result (client repair)")
	msg.ToolUseResult = tr
	return msg
}
