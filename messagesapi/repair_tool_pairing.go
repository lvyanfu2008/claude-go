package messagesapi

import (
	"encoding/json"

	"goc/ccb-engine/diaglog"
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
		j := i + 1
		var userRun []types.Message
		for j < len(messages) {
			u := messagerow.NormalizeMessageJSON(messages[j])
			if u.Type != types.MessageTypeUser {
				break
			}
			userRun = append(userRun, u)
			j++
		}
		if len(userRun) == 0 {
			diaglog.LineOrStderr("[tool-pairing] inserted user message: %d synthetic tool_result(s) (assistant=%s tool_use_ids=%v)",
				len(required), asstUUID, required)
			out = append(out, syntheticUserWithToolResults(required, asstUUID))
			continue
		}
		if UserMessagesCoverAssistantToolUses(m, userRun) {
			out = append(out, userRun...)
			i = j - 1
			continue
		}
		patchedRun, nAdded, missLog, err := patchConsecutiveUserRunForMissingToolResults(userRun, required, asstUUID)
		if err != nil {
			return nil, err
		}
		if nAdded > 0 {
			diaglog.LineOrStderr("[tool-pairing] patched user message: added %d synthetic tool_result(s) (assistant=%s tool_use_ids=%v)",
				nAdded, asstUUID, missLog)
		}
		out = append(out, patchedRun...)
		i = j - 1
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

// patchConsecutiveUserRunForMissingToolResults treats all consecutive user messages after
// an assistant as one run. If tool_result ids are split across those rows, we do not add
// synthetics (the run is complete). If something is still missing, we append synthetic
// tool_result blocks to the last user in the run only.
func patchConsecutiveUserRunForMissingToolResults(userRun []types.Message, required []string, asstUUID string) ([]types.Message, int, []string, error) {
	have := make(map[string]struct{})
	for i := range userRun {
		for _, id := range toolResultIDsInUserMessage(&userRun[i]) {
			have[id] = struct{}{}
		}
	}
	var missing []string
	for _, id := range required {
		if _, ok := have[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return userRun, 0, nil, nil
	}
	last := userRun[len(userRun)-1]
	patched, nAdded, err := patchUserAppendingMissingToolResultBlocks(last, missing, asstUUID)
	if err != nil {
		return nil, 0, nil, err
	}
	out := append(append([]types.Message{}, userRun[:len(userRun)-1]...), patched)
	return out, nAdded, missing, nil
}

func patchUserAppendingMissingToolResultBlocks(m types.Message, missing []string, sourceAsstUUID string) (types.Message, int, error) {
	m2, err := cloneMessage(m)
	if err != nil {
		return types.Message{}, 0, err
	}
	if err := ensureInnerFromContent(&m2); err != nil {
		return types.Message{}, 0, err
	}
	inner, err := getInner(&m2)
	if err != nil {
		return types.Message{}, 0, err
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return types.Message{}, 0, err
	}
	nAdded := len(missing)
	for _, id := range missing {
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
		return types.Message{}, 0, err
	}
	inner.Content = raw
	if err := setInner(&m2, inner); err != nil {
		return types.Message{}, 0, err
	}
	syncTopLevelContent(&m2)
	if sourceAsstUUID != "" {
		m2.SourceToolAssistantUUID = &sourceAsstUUID
	}
	return m2, nAdded, nil
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
