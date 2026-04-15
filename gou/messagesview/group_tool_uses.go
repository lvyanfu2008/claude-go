package messagesview

import (
	"encoding/json"
	"fmt"

	"goc/types"
)

var toolsWithGrouping = map[string]bool{
	"Agent": true,
}

type toolUseInfo struct {
	MessageID string
	ToolUseID string
	ToolName  string
}

func getToolUseInfo(msg types.Message) *toolUseInfo {
	if msg.Type != types.MessageTypeAssistant {
		return nil
	}

	// Try to get message ID
	msgID := ""
	if msg.MessageID != nil && *msg.MessageID != "" {
		msgID = *msg.MessageID
	} else if len(msg.Message) > 0 {
		var envelope struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(msg.Message, &envelope) == nil {
			msgID = envelope.ID
		}
	}
	// Fallback to msg.UUID if we still don't have an ID
	if msgID == "" {
		msgID = msg.UUID
	}

	var content []json.RawMessage
	if len(msg.Content) > 0 {
		if json.Unmarshal(msg.Content, &content) != nil {
			return nil
		}
	} else if len(msg.Message) > 0 {
		var inner struct {
			Content []json.RawMessage `json:"content"`
		}
		if json.Unmarshal(msg.Message, &inner) == nil {
			content = inner.Content
		}
	}

	if len(content) == 0 {
		return nil
	}

	var firstBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if json.Unmarshal(content[0], &firstBlock) != nil || firstBlock.Type != "tool_use" {
		return nil
	}

	return &toolUseInfo{
		MessageID: msgID,
		ToolUseID: firstBlock.ID,
		ToolName:  firstBlock.Name,
	}
}

// ApplyGrouping groups tool uses by message.id if the tool supports grouped rendering (Agent).
// Only groups 2+ tools of the same type from the same message.
// Also collects corresponding tool_results and attaches them to the grouped message.
// When verbose is true, skips grouping so messages render at original positions.
func ApplyGrouping(messages []types.Message, verbose bool) []types.Message {
	if verbose || len(messages) == 0 {
		return messages
	}

	// First pass: group tool uses by message.id + tool name
	groups := make(map[string][]types.Message)
	groupKeys := []string{} // to preserve order of discovery

	for _, msg := range messages {
		info := getToolUseInfo(msg)
		if info != nil && toolsWithGrouping[info.ToolName] {
			key := info.MessageID + ":" + info.ToolName
			if _, ok := groups[key]; !ok {
				groupKeys = append(groupKeys, key)
			}
			groups[key] = append(groups[key], msg)
		}
	}

	// Identify valid groups (2+ items) and collect their tool use IDs
	validGroups := make(map[string][]types.Message)
	groupedToolUseIds := make(map[string]bool)

	for _, key := range groupKeys {
		group := groups[key]
		if len(group) >= 2 {
			validGroups[key] = group
			for _, msg := range group {
				info := getToolUseInfo(msg)
				if info != nil {
					groupedToolUseIds[info.ToolUseID] = true
				}
			}
		}
	}

	// Collect result messages for grouped tool_uses
	resultsByToolUseId := make(map[string]types.Message)

	for _, msg := range messages {
		if msg.Type == types.MessageTypeUser && len(msg.Content) > 0 {
			var blocks []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
			}
			if json.Unmarshal(msg.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "tool_result" && groupedToolUseIds[block.ToolUseID] {
						resultsByToolUseId[block.ToolUseID] = msg
					}
				}
			}
		}
	}

	// Second pass: build output, emitting each group only once
	result := make([]types.Message, 0, len(messages))
	emittedGroups := make(map[string]bool)

	for _, msg := range messages {
		info := getToolUseInfo(msg)

		if info != nil {
			key := info.MessageID + ":" + info.ToolName
			group, ok := validGroups[key]

			if ok {
				if !emittedGroups[key] {
					emittedGroups[key] = true
					firstMsg := group[0]

					// Collect results for this group
					var results []types.Message
					for _, assistantMsg := range group {
						astInfo := getToolUseInfo(assistantMsg)
						if astInfo != nil {
							if resMsg, ok := resultsByToolUseId[astInfo.ToolUseID]; ok {
								results = append(results, resMsg)
							}
						}
					}

					msgIDRef := info.MessageID
					groupedMessage := types.Message{
						Type:           types.MessageTypeGroupedToolUse,
						ToolName:       info.ToolName,
						Messages:       group,
						Results:        results,
						DisplayMessage: &firstMsg,
						UUID:           fmt.Sprintf("grouped-%s", firstMsg.UUID),
						Timestamp:      firstMsg.Timestamp,
						MessageID:      &msgIDRef,
					}
					result = append(result, groupedMessage)
				}
				continue
			}
		}

		// Skip user messages whose tool_results are all grouped
		if msg.Type == types.MessageTypeUser && len(msg.Content) > 0 {
			var blocks []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
			}
			if json.Unmarshal(msg.Content, &blocks) == nil {
				hasToolResult := false
				allGrouped := true
				for _, block := range blocks {
					if block.Type == "tool_result" {
						hasToolResult = true
						if !groupedToolUseIds[block.ToolUseID] {
							allGrouped = false
						}
					}
				}
				if hasToolResult && allGrouped {
					continue
				}
			}
		}

		result = append(result, msg)
	}

	return result
}
