package messagesapi

import (
	"encoding/json"
	"goc/gou/messagerow"
	"goc/types"
)

// NormalizeMessagesForAPI mirrors src/utils/messages.ts normalizeMessagesForAPI.
func NormalizeMessagesForAPI(messages []types.Message, tools []ToolSpec, opts Options) ([]types.Message, error) {
	prep := make([]types.Message, len(messages))
	for i := range messages {
		m := messagerow.NormalizeMessageJSON(messages[i])
		if m.Type == types.MessageTypeUser || m.Type == types.MessageTypeAssistant {
			_ = ensureInnerFromContent(&m)
		}
		prep[i] = m
	}
	messages = prep

	availableToolNames := make(map[string]struct{}, len(tools))
	for _, t := range tools {
		availableToolNames[t.Name] = struct{}{}
	}

	reorderedMessages := reorderAttachmentsForAPI(messages)
	var filtered []types.Message
	for _, m := range reorderedMessages {
		if (m.Type == types.MessageTypeUser || m.Type == types.MessageTypeAssistant) && isTruthy(m.IsVirtual) {
			continue
		}
		filtered = append(filtered, m)
	}
	reorderedMessages = filtered

	errorToBlockTypes := errorToBlockTypes(opts.NonInteractive)
	stripTargets := make(map[string]map[string]struct{})
	for i := 0; i < len(reorderedMessages); i++ {
		msg := reorderedMessages[i]
		if !isSyntheticApiErrorMessage(msg) {
			continue
		}
		errorText := assistantFirstTextBlock(&msg)
		if errorText == "" {
			continue
		}
		blockTypesToStrip, ok := errorToBlockTypes[errorText]
		if !ok {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			candidate := reorderedMessages[j]
			if candidate.Type == types.MessageTypeUser && isTruthy(candidate.IsMeta) {
				existing := stripTargets[candidate.UUID]
				if existing == nil {
					existing = make(map[string]struct{})
					for k := range blockTypesToStrip {
						existing[k] = struct{}{}
					}
					stripTargets[candidate.UUID] = existing
				} else {
					for k := range blockTypesToStrip {
						existing[k] = struct{}{}
					}
				}
				break
			}
			if isSyntheticApiErrorMessage(candidate) {
				continue
			}
			break
		}
	}

	var result []types.Message
	uuidGen := randomUUID
	for _, message := range reorderedMessages {
		if message.Type == types.MessageTypeProgress {
			continue
		}
		if message.Type == types.MessageTypeSystem && !isSystemLocalCommandMessage(message) {
			continue
		}
		if isSyntheticApiErrorMessage(message) {
			continue
		}

		switch message.Type {
		case types.MessageTypeSystem:
			contentRaw := systemMessageContent(&message)
			var contentForUser json.RawMessage
			if len(contentRaw) > 0 {
				contentForUser = contentRaw
			} else {
				contentForUser = json.RawMessage(`""`)
			}
			ts := ""
			if message.Timestamp != nil {
				ts = *message.Timestamp
			}
			userMsg := createUserMessageFromContent(contentForUser, message.UUID, ts, false)
			syncTopLevelContent(&userMsg)
			if len(result) > 0 {
				last := &result[len(result)-1]
				if last.Type == types.MessageTypeUser {
					merged, err := mergeUserMessages(*last, userMsg, opts)
					if err != nil {
						return nil, err
					}
					syncTopLevelContent(&merged)
					result[len(result)-1] = merged
					continue
				}
			}
			result = append(result, userMsg)

		case types.MessageTypeUser:
			normalizedMessage := message
			var err error
			if !opts.ToolSearchEnabled {
				normalizedMessage, err = stripToolReferenceBlocksFromUserMessage(normalizedMessage)
				if err != nil {
					return nil, err
				}
			} else {
				normalizedMessage, err = stripUnavailableToolReferencesFromUserMessage(normalizedMessage, availableToolNames)
				if err != nil {
					return nil, err
				}
			}

			typesToStrip := stripTargets[normalizedMessage.UUID]
			if typesToStrip != nil && isTruthy(normalizedMessage.IsMeta) {
				inner, err := getInner(&normalizedMessage)
				if err != nil {
					return nil, err
				}
				blocks, err := parseContentArrayOrString(inner.Content)
				if err != nil {
					return nil, err
				}
				if len(blocks) > 0 {
					var filteredBlocks []map[string]any
					for _, block := range blocks {
						bt, _ := block["type"].(string)
						if _, drop := typesToStrip[bt]; drop {
							continue
						}
						filteredBlocks = append(filteredBlocks, block)
					}
					if len(filteredBlocks) == 0 {
						continue
					}
					if len(filteredBlocks) < len(blocks) {
						raw, err := marshalContentBlocks(filteredBlocks)
						if err != nil {
							return nil, err
						}
						inner.Content = raw
						if err := setInner(&normalizedMessage, inner); err != nil {
							return nil, err
						}
						syncTopLevelContent(&normalizedMessage)
					}
				}
			}

			if !opts.ToolrefDeferJ8m {
				inner, err := getInner(&normalizedMessage)
				if err != nil {
					return nil, err
				}
				blocks, err := parseContentArrayOrString(inner.Content)
				if err != nil {
					return nil, err
				}
				if len(blocks) > 0 && !textBlocksStartWithPrefix(blocks, toolReferenceTurnBoundary) && contentHasToolReference(blocks) {
					blocks = append(blocks, map[string]any{"type": "text", "text": toolReferenceTurnBoundary})
					raw, err := marshalContentBlocks(blocks)
					if err != nil {
						return nil, err
					}
					inner.Content = raw
					if err := setInner(&normalizedMessage, inner); err != nil {
						return nil, err
					}
					syncTopLevelContent(&normalizedMessage)
				}
			}

			if len(result) > 0 {
				last := &result[len(result)-1]
				if last.Type == types.MessageTypeUser {
					merged, err := mergeUserMessages(*last, normalizedMessage, opts)
					if err != nil {
						return nil, err
					}
					syncTopLevelContent(&merged)
					result[len(result)-1] = merged
					continue
				}
			}
			syncTopLevelContent(&normalizedMessage)
			result = append(result, normalizedMessage)

		case types.MessageTypeAssistant:
			normalizedMessage, err := normalizeAssistantForAPI(message, tools, opts)
			if err != nil {
				return nil, err
			}
			syncTopLevelContent(&normalizedMessage)
			mergedInto := false
			for i := len(result) - 1; i >= 0; i-- {
				msg := result[i]
				if msg.Type != types.MessageTypeAssistant && !isToolResultMessage(msg) {
					break
				}
				if msg.Type == types.MessageTypeAssistant {
					innerN, _ := getInner(&normalizedMessage)
					innerM, _ := getInner(&msg)
					if innerN.ID != "" && innerN.ID == innerM.ID {
						merged, err := mergeAssistantMessages(msg, normalizedMessage)
						if err != nil {
							return nil, err
						}
						syncTopLevelContent(&merged)
						result[i] = merged
						mergedInto = true
						break
					}
				}
			}
			if !mergedInto {
				result = append(result, normalizedMessage)
			}

		case types.MessageTypeAttachment:
			rawAttachmentMessage, err := normalizeAttachmentForAPI(message.Attachment, opts, uuidGen)
			if err != nil {
				return nil, err
			}
			attachmentMessage := rawAttachmentMessage
			if opts.ChairSermon {
				for i := range attachmentMessage {
					attachmentMessage[i] = ensureSystemReminderWrap(attachmentMessage[i])
				}
			}
			if len(result) > 0 {
				last := &result[len(result)-1]
				if last.Type == types.MessageTypeUser {
					cur := *last
					for _, c := range attachmentMessage {
						cur, err = mergeUserMessagesAndToolResults(cur, c, opts)
						if err != nil {
							return nil, err
						}
						syncTopLevelContent(&cur)
					}
					result[len(result)-1] = cur
					continue
				}
			}
			for _, c := range attachmentMessage {
				cc := c
				syncTopLevelContent(&cc)
				result = append(result, cc)
			}
		}
	}

	var relocated []types.Message
	var err error
	if opts.ToolrefDeferJ8m {
		relocated, err = relocateToolReferenceSiblings(result)
		if err != nil {
			return nil, err
		}
	} else {
		relocated = result
	}

	withFilteredOrphans, err := filterOrphanedThinkingOnlyMessages(relocated)
	if err != nil {
		return nil, err
	}
	withFilteredThinking, err := filterTrailingThinkingFromLastAssistant(withFilteredOrphans)
	if err != nil {
		return nil, err
	}
	withFilteredWhitespace, err := filterWhitespaceOnlyAssistantMessages(withFilteredThinking, opts)
	if err != nil {
		return nil, err
	}
	withNonEmpty, err := ensureNonEmptyAssistantContent(withFilteredWhitespace)
	if err != nil {
		return nil, err
	}

	var smooshed []types.Message
	if opts.ChairSermon {
		mergedAdj, err := mergeAdjacentUserMessages(withNonEmpty, opts)
		if err != nil {
			return nil, err
		}
		smooshed, err = smooshSystemReminderSiblings(mergedAdj)
		if err != nil {
			return nil, err
		}
	} else {
		smooshed = withNonEmpty
	}

	sanitized, err := sanitizeErrorToolResultContent(smooshed)
	if err != nil {
		return nil, err
	}

	if opts.CompactAllTextUserContent {
		sanitized, err = collapseAllTextUserContentBlocks(sanitized)
		if err != nil {
			return nil, err
		}
	}

	if opts.HistorySnip && !opts.TestMode {
		for i := range sanitized {
			if sanitized[i].Type != types.MessageTypeUser {
				continue
			}
			upd, err := appendMessageTagToUserMessage(sanitized[i])
			if err != nil {
				return nil, err
			}
			syncTopLevelContent(&upd)
			sanitized[i] = upd
		}
	}

	if !opts.SkipImageValidation {
		if err := validateImagesForAPI(sanitized); err != nil {
			return nil, err
		}
	}

	for i := range sanitized {
		syncTopLevelContent(&sanitized[i])
	}

	return sanitized, nil
}

func syncTopLevelContent(m *types.Message) {
	inner, err := getInner(m)
	if err != nil {
		return
	}
	m.Content = inner.Content
}

func textBlocksStartWithPrefix(blocks []map[string]any, prefix string) bool {
	for _, b := range blocks {
		if t, _ := b["type"].(string); t == "text" {
			tx, _ := b["text"].(string)
			if len(tx) >= len(prefix) && tx[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

func normalizeAssistantForAPI(message types.Message, tools []ToolSpec, opts Options) (types.Message, error) {
	m, err := cloneMessage(message)
	if err != nil {
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
	toolSearchEnabled := opts.ToolSearchEnabled
	var outBlocks []map[string]any
	for _, block := range blocks {
		t, _ := block["type"].(string)
		if t != "tool_use" {
			outBlocks = append(outBlocks, block)
			continue
		}
		toolUseBlk := block
		name, _ := toolUseBlk["name"].(string)
		tool := findToolByName(tools, name)
		canonicalName := name
		if tool != nil {
			canonicalName = tool.Name
		}
		input := toolUseBlk["input"]
		normalizedInput := normalizeToolInputForAPI(canonicalName, input)
		if toolSearchEnabled {
			nb := cloneMapShallow(toolUseBlk)
			nb["name"] = canonicalName
			nb["input"] = normalizedInput
			outBlocks = append(outBlocks, nb)
		} else {
			outBlocks = append(outBlocks, map[string]any{
				"type":  "tool_use",
				"id":    toolUseBlk["id"],
				"name":  canonicalName,
				"input": normalizedInput,
			})
		}
	}
	if !toolSearchEnabled {
		outBlocks = stripCallerFromToolUseBlocks(outBlocks)
	}
	raw, err := marshalContentBlocks(outBlocks)
	if err != nil {
		return types.Message{}, err
	}
	inner.Content = raw
	if err := setInner(&m, inner); err != nil {
		return types.Message{}, err
	}
	return m, nil
}
