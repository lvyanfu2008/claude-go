package messagesapi

import (
	"encoding/json"
	"strings"

	"goc/types"
)

func relocateToolReferenceSiblings(messages []types.Message) ([]types.Message, error) {
	result := make([]types.Message, len(messages))
	for i := range messages {
		c, err := cloneMessage(messages[i])
		if err != nil {
			return nil, err
		}
		result[i] = c
	}

	for i := 0; i < len(result); i++ {
		msg := &result[i]
		if msg.Type != types.MessageTypeUser {
			continue
		}
		inner, err := getInner(msg)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		if !contentHasToolReference(blocks) {
			continue
		}
		var textSiblings []map[string]any
		var nonText []map[string]any
		for _, b := range blocks {
			if t, _ := b["type"].(string); t == "text" {
				textSiblings = append(textSiblings, b)
			} else {
				nonText = append(nonText, b)
			}
		}
		if len(textSiblings) == 0 {
			continue
		}
		targetIdx := -1
		for j := i + 1; j < len(result); j++ {
			cand := &result[j]
			if cand.Type != types.MessageTypeUser {
				continue
			}
			innerC, err := getInner(cand)
			if err != nil {
				return nil, err
			}
			cc, err := parseContentArrayOrString(innerC.Content)
			if err != nil {
				return nil, err
			}
			hasTR := false
			for _, b := range cc {
				if t, _ := b["type"].(string); t == "tool_result" {
					hasTR = true
					break
				}
			}
			if !hasTR {
				continue
			}
			if contentHasToolReference(cc) {
				continue
			}
			targetIdx = j
			break
		}
		if targetIdx == -1 {
			continue
		}
		newSrc, err := marshalContentBlocks(nonText)
		if err != nil {
			return nil, err
		}
		inner.Content = newSrc
		if err := setInner(msg, inner); err != nil {
			return nil, err
		}

		tgt := &result[targetIdx]
		innerT, err := getInner(tgt)
		if err != nil {
			return nil, err
		}
		tgtBlocks, err := parseContentArrayOrString(innerT.Content)
		if err != nil {
			return nil, err
		}
		tgtBlocks = append(tgtBlocks, textSiblings...)
		raw, err := marshalContentBlocks(tgtBlocks)
		if err != nil {
			return nil, err
		}
		innerT.Content = raw
		if err := setInner(tgt, innerT); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func wrapInSystemReminder(content string) string {
	return "<system-reminder>\n" + content + "\n</system-reminder>"
}

func ensureSystemReminderWrap(msg types.Message) types.Message {
	m := msg
	inner, err := getInner(&m)
	if err != nil {
		return msg
	}
	if len(inner.Content) == 0 {
		return msg
	}
	var s string
	if err := json.Unmarshal(inner.Content, &s); err == nil {
		if strings.HasPrefix(s, "<system-reminder>") {
			return m
		}
		raw, _ := json.Marshal(wrapInSystemReminder(s))
		inner.Content = raw
		_ = setInner(&m, inner)
		m.Content = raw
		return m
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) == 0 {
		return m
	}
	changed := false
	for i, b := range blocks {
		if t, _ := b["type"].(string); t != "text" {
			continue
		}
		tx, _ := b["text"].(string)
		if strings.HasPrefix(tx, "<system-reminder>") {
			continue
		}
		b["text"] = wrapInSystemReminder(tx)
		blocks[i] = b
		changed = true
	}
	if !changed {
		return m
	}
	raw, err := marshalContentBlocks(blocks)
	if err != nil {
		return m
	}
	inner.Content = raw
	_ = setInner(&m, inner)
	m.Content = raw
	return m
}

func smooshSystemReminderSiblings(messages []types.Message) ([]types.Message, error) {
	out := make([]types.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Type != types.MessageTypeUser {
			out = append(out, msg)
			continue
		}
		m := msg
		inner, err := getInner(&m)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		hasTR := false
		for _, b := range blocks {
			if t, _ := b["type"].(string); t == "tool_result" {
				hasTR = true
				break
			}
		}
		if !hasTR {
			out = append(out, m)
			continue
		}
		var srText []map[string]any
		var kept []map[string]any
		for _, b := range blocks {
			if t, _ := b["type"].(string); t == "text" {
				tx, _ := b["text"].(string)
				if strings.HasPrefix(tx, "<system-reminder>") {
					srText = append(srText, b)
					continue
				}
			}
			kept = append(kept, b)
		}
		if len(srText) == 0 {
			out = append(out, m)
			continue
		}
		lastTrIdx := -1
		for i := len(kept) - 1; i >= 0; i-- {
			if t, _ := kept[i]["type"].(string); t == "tool_result" {
				lastTrIdx = i
				break
			}
		}
		if lastTrIdx < 0 {
			out = append(out, m)
			continue
		}
		lastTr := kept[lastTrIdx]
		smooshed, err := smooshIntoToolResult(lastTr, srText)
		if err != nil || smooshed == nil {
			out = append(out, m)
			continue
		}
		newKept := make([]map[string]any, 0, len(kept))
		newKept = append(newKept, kept[:lastTrIdx]...)
		newKept = append(newKept, smooshed)
		newKept = append(newKept, kept[lastTrIdx+1:]...)
		raw, err := marshalContentBlocks(newKept)
		if err != nil {
			return nil, err
		}
		inner.Content = raw
		if err := setInner(&m, inner); err != nil {
			return nil, err
		}
		m.Content = raw
		out = append(out, m)
	}
	return out, nil
}

func sanitizeErrorToolResultContent(messages []types.Message) ([]types.Message, error) {
	out := make([]types.Message, len(messages))
	for i := range messages {
		c, err := cloneMessage(messages[i])
		if err != nil {
			return nil, err
		}
		out[i] = c
	}
	for i := range out {
		msg := &out[i]
		if msg.Type != types.MessageTypeUser {
			continue
		}
		inner, err := getInner(msg)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		changed := false
		for j, b := range blocks {
			if t, _ := b["type"].(string); t != "tool_result" {
				continue
			}
			isErr, _ := b["is_error"].(bool)
			if !isErr {
				continue
			}
			raw := b["content"]
			arr, ok := raw.([]any)
			if !ok {
				continue
			}
			allText := true
			for _, c := range arr {
				m, ok := c.(map[string]any)
				if !ok {
					allText = false
					break
				}
				if typ, _ := m["type"].(string); typ != "text" {
					allText = false
					break
				}
			}
			if allText {
				continue
			}
			changed = true
			var texts []string
			for _, c := range arr {
				m, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := m["type"].(string); typ == "text" {
					tx, _ := m["text"].(string)
					texts = append(texts, tx)
				}
			}
			var newContent any
			if len(texts) > 0 {
				newContent = []any{map[string]any{"type": "text", "text": strings.Join(texts, "\n\n")}}
			} else {
				newContent = []any{}
			}
			b["content"] = newContent
			blocks[j] = b
		}
		if !changed {
			continue
		}
		raw, err := marshalContentBlocks(blocks)
		if err != nil {
			return nil, err
		}
		inner.Content = raw
		if err := setInner(msg, inner); err != nil {
			return nil, err
		}
		msg.Content = raw
	}
	return out, nil
}

func isThinkingBlock(block map[string]any) bool {
	t, _ := block["type"].(string)
	return t == "thinking" || t == "redacted_thinking"
}

func filterTrailingThinkingFromLastAssistant(messages []types.Message) ([]types.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}
	last := messages[len(messages)-1]
	if last.Type != types.MessageTypeAssistant {
		return messages, nil
	}
	inner, err := getInner(&last)
	if err != nil {
		return nil, err
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) == 0 {
		return messages, nil
	}
	lastValid := len(blocks) - 1
	for lastValid >= 0 {
		if !isThinkingBlock(blocks[lastValid]) {
			break
		}
		lastValid--
	}
	if lastValid == len(blocks)-1 {
		return messages, nil
	}
	var filtered []map[string]any
	if lastValid < 0 {
		filtered = []map[string]any{{"type": "text", "text": "[No message content]"}}
	} else {
		filtered = blocks[:lastValid+1]
	}
	raw, err := marshalContentBlocks(filtered)
	if err != nil {
		return nil, err
	}
	inner.Content = raw
	out := make([]types.Message, len(messages))
	copy(out, messages)
	lastMsg := out[len(out)-1]
	if err := setInner(&lastMsg, inner); err != nil {
		return nil, err
	}
	lastMsg.Content = raw
	out[len(out)-1] = lastMsg
	return out, nil
}

func hasOnlyWhitespaceTextContent(blocks []map[string]any) bool {
	if len(blocks) == 0 {
		return false
	}
	for _, b := range blocks {
		t, _ := b["type"].(string)
		if t != "text" {
			return false
		}
		tx, _ := b["text"].(string)
		if strings.TrimSpace(tx) != "" {
			return false
		}
	}
	return true
}

func filterWhitespaceOnlyAssistantMessages(messages []types.Message, opts Options) ([]types.Message, error) {
	filtered := make([]types.Message, 0, len(messages))
	for _, message := range messages {
		if message.Type != types.MessageTypeAssistant {
			filtered = append(filtered, message)
			continue
		}
		inner, err := getInner(&message)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		if len(blocks) == 0 {
			filtered = append(filtered, message)
			continue
		}
		if hasOnlyWhitespaceTextContent(blocks) {
			continue
		}
		filtered = append(filtered, message)
	}
	if len(filtered) == len(messages) {
		return messages, nil
	}
	return mergeAdjacentUserMessages(filtered, opts)
}

func ensureNonEmptyAssistantContent(messages []types.Message) ([]types.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}
	out := make([]types.Message, len(messages))
	copy(out, messages)
	changed := false
	for i := range out[:len(out)-1] {
		msg := &out[i]
		if msg.Type != types.MessageTypeAssistant {
			continue
		}
		inner, err := getInner(msg)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		if len(blocks) != 0 {
			continue
		}
		changed = true
		placeholder, err := marshalContentBlocks([]map[string]any{{"type": "text", "text": noContentMessage}})
		if err != nil {
			return nil, err
		}
		inner.Content = placeholder
		if err := setInner(msg, inner); err != nil {
			return nil, err
		}
		msg.Content = placeholder
	}
	if !changed {
		return messages, nil
	}
	return out, nil
}

func filterOrphanedThinkingOnlyMessages(messages []types.Message) ([]types.Message, error) {
	idsWithNonThinking := make(map[string]struct{})
	for _, msg := range messages {
		if msg.Type != types.MessageTypeAssistant {
			continue
		}
		inner, err := getInner(&msg)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		hasNonThinking := false
		for _, b := range blocks {
			if !isThinkingBlock(b) {
				hasNonThinking = true
				break
			}
		}
		if hasNonThinking && inner.ID != "" {
			idsWithNonThinking[inner.ID] = struct{}{}
		}
	}
	var filtered []types.Message
	for _, msg := range messages {
		if msg.Type != types.MessageTypeAssistant {
			filtered = append(filtered, msg)
			continue
		}
		inner, err := getInner(&msg)
		if err != nil {
			return nil, err
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			return nil, err
		}
		if len(blocks) == 0 {
			filtered = append(filtered, msg)
			continue
		}
		allThinking := true
		for _, b := range blocks {
			if !isThinkingBlock(b) {
				allThinking = false
				break
			}
		}
		if !allThinking {
			filtered = append(filtered, msg)
			continue
		}
		if inner.ID != "" {
			if _, ok := idsWithNonThinking[inner.ID]; ok {
				filtered = append(filtered, msg)
				continue
			}
		}
		// drop orphaned
	}
	return filtered, nil
}

func appendMessageTagToUserMessage(msg types.Message) (types.Message, error) {
	if isTruthy(msg.IsMeta) {
		return msg, nil
	}
	m := msg
	inner, err := getInner(&m)
	if err != nil {
		return msg, err
	}
	tag := "\n[id:" + deriveShortMessageId(msg.UUID) + "]"
	if len(inner.Content) == 0 {
		return msg, nil
	}
	var s string
	if err := json.Unmarshal(inner.Content, &s); err == nil {
		raw, _ := json.Marshal(s + tag)
		inner.Content = raw
		if err := setInner(&m, inner); err != nil {
			return msg, err
		}
		m.Content = raw
		return m, nil
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		return msg, err
	}
	if len(blocks) == 0 {
		return msg, nil
	}
	lastTextIdx := -1
	for i := len(blocks) - 1; i >= 0; i-- {
		if t, _ := blocks[i]["type"].(string); t == "text" {
			lastTextIdx = i
			break
		}
	}
	if lastTextIdx < 0 {
		return msg, nil
	}
	tx, _ := blocks[lastTextIdx]["text"].(string)
	blocks[lastTextIdx]["text"] = tx + tag
	raw, err := marshalContentBlocks(blocks)
	if err != nil {
		return msg, err
	}
	inner.Content = raw
	if err := setInner(&m, inner); err != nil {
		return msg, err
	}
	m.Content = raw
	return m, nil
}

func isToolResultMessage(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	inner, err := getInner(&msg)
	if err != nil || len(inner.Content) == 0 {
		return false
	}
	blocks, err := parseContentArrayOrString(inner.Content)
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
