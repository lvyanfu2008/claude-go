package messagesapi

import (
	"encoding/json"

	"goc/types"
)

func isToolReferenceBlock(obj any) bool {
	m, ok := obj.(map[string]any)
	if !ok || m == nil {
		return false
	}
	t, _ := m["type"].(string)
	return t == "tool_reference"
}

func contentHasToolReference(blocks []map[string]any) bool {
	for _, block := range blocks {
		if t, _ := block["type"].(string); t != "tool_result" {
			continue
		}
		raw, ok := block["content"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, c := range arr {
			if isToolReferenceBlock(c) {
				return true
			}
		}
	}
	return false
}

// stripToolReferenceBlocksFromUserMessage mirrors TS when tool search disabled.
func stripToolReferenceBlocksFromUserMessage(message types.Message) (types.Message, error) {
	m := message
	inner, err := getInner(&m)
	if err != nil {
		return message, err
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return message, err
	}
	if len(blocks) == 0 {
		return message, nil
	}
	if !userBlocksHaveToolReference(blocks) {
		return message, nil
	}
	newBlocks := make([]map[string]any, len(blocks))
	copy(newBlocks, blocks)
	for i, block := range newBlocks {
		if t, _ := block["type"].(string); t != "tool_result" {
			continue
		}
		raw, ok := block["content"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		filtered := filterOutToolReferences(arr)
		if len(filtered) == 0 {
			block["content"] = []any{map[string]any{"type": "text", "text": "[Tool references removed - tool search not enabled]"}}
		} else {
			block["content"] = filtered
		}
		newBlocks[i] = block
	}
	raw, err := marshalContentBlocks(newBlocks)
	if err != nil {
		return message, err
	}
	inner.Content = raw
	if err := setInner(&m, inner); err != nil {
		return message, err
	}
	return m, nil
}

func userBlocksHaveToolReference(blocks []map[string]any) bool {
	for _, block := range blocks {
		if t, _ := block["type"].(string); t != "tool_result" {
			continue
		}
		raw, ok := block["content"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, c := range arr {
			if isToolReferenceBlock(c) {
				return true
			}
		}
	}
	return false
}

func filterOutToolReferences(arr []any) []any {
	var out []any
	for _, c := range arr {
		if !isToolReferenceBlock(c) {
			out = append(out, c)
		}
	}
	return out
}

func stripUnavailableToolReferencesFromUserMessage(message types.Message, availableToolNames map[string]struct{}) (types.Message, error) {
	m := message
	inner, err := getInner(&m)
	if err != nil {
		return message, err
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return message, err
	}
	if !userBlocksHaveUnavailableToolRef(blocks, availableToolNames) {
		return message, nil
	}
	newBlocks := make([]map[string]any, len(blocks))
	copy(newBlocks, blocks)
	for i, block := range newBlocks {
		if t, _ := block["type"].(string); t != "tool_result" {
			continue
		}
		raw, ok := block["content"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		filtered := filterToolRefsByAvailability(arr, availableToolNames)
		if len(filtered) == 0 {
			block["content"] = []any{map[string]any{"type": "text", "text": "[Tool references removed - tools no longer available]"}}
		} else {
			block["content"] = filtered
		}
		newBlocks[i] = block
	}
	raw, err := marshalContentBlocks(newBlocks)
	if err != nil {
		return message, err
	}
	inner.Content = raw
	if err := setInner(&m, inner); err != nil {
		return message, err
	}
	return m, nil
}

func userBlocksHaveUnavailableToolRef(blocks []map[string]any, availableToolNames map[string]struct{}) bool {
	for _, block := range blocks {
		if t, _ := block["type"].(string); t != "tool_result" {
			continue
		}
		raw, ok := block["content"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, c := range arr {
			m, ok := c.(map[string]any)
			if !ok || !isToolReferenceBlock(m) {
				continue
			}
			tn, _ := m["tool_name"].(string)
			if tn == "" {
				continue
			}
			canon := normalizeLegacyToolName(tn)
			if _, ok := availableToolNames[canon]; !ok {
				return true
			}
		}
	}
	return false
}

func filterToolRefsByAvailability(arr []any, availableToolNames map[string]struct{}) []any {
	var out []any
	for _, c := range arr {
		m, ok := c.(map[string]any)
		if !ok || !isToolReferenceBlock(m) {
			out = append(out, c)
			continue
		}
		tn, _ := m["tool_name"].(string)
		if tn == "" {
			out = append(out, c)
			continue
		}
		canon := normalizeLegacyToolName(tn)
		if _, ok := availableToolNames[canon]; ok {
			out = append(out, c)
		}
	}
	return out
}

func toolUseBlockHasCaller(block map[string]any) bool {
	if t, _ := block["type"].(string); t != "tool_use" {
		return false
	}
	_, ok := block["caller"]
	return ok
}

func stripCallerFromToolUseBlocks(blocks []map[string]any) []map[string]any {
	out := make([]map[string]any, len(blocks))
	for i, b := range blocks {
		if t, _ := b["type"].(string); t != "tool_use" {
			out[i] = b
			continue
		}
		if !toolUseBlockHasCaller(b) {
			out[i] = b
			continue
		}
		nb := map[string]any{
			"type":  "tool_use",
			"id":    b["id"],
			"name":  b["name"],
			"input": b["input"],
		}
		out[i] = nb
	}
	return out
}

// assistantFirstTextBlock extracts first text block string from assistant message (synthetic errors).
func assistantFirstTextBlock(m *types.Message) string {
	inner, err := getInner(m)
	if err != nil {
		return ""
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) == 0 {
		return ""
	}
	t, _ := blocks[0]["type"].(string)
	if t != "text" {
		return ""
	}
	s, _ := blocks[0]["text"].(string)
	return s
}

func isSyntheticApiErrorMessage(m types.Message) bool {
	if m.Type != types.MessageTypeAssistant {
		return false
	}
	if !isTruthy(m.IsApiErrorMessage) {
		return false
	}
	inner, err := getInner(&m)
	if err != nil {
		return false
	}
	return inner.Model == syntheticModel
}

func isSystemLocalCommandMessage(m types.Message) bool {
	if m.Type != types.MessageTypeSystem {
		return false
	}
	if m.Subtype == nil {
		return false
	}
	return *m.Subtype == "local_command"
}

// systemMessageContent returns top-level content for system messages (JSON string or raw).
func systemMessageContent(m *types.Message) json.RawMessage {
	if len(m.Content) > 0 {
		return m.Content
	}
	return nil
}
