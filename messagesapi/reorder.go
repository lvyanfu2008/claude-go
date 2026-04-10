package messagesapi

import (
	"goc/types"
)

// reorderAttachmentsForAPI mirrors src/utils/messages.ts reorderAttachmentsForAPI.
func reorderAttachmentsForAPI(messages []types.Message) []types.Message {
	var result []types.Message
	var pendingAttachments []types.Message

	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Type == types.MessageTypeAttachment {
			pendingAttachments = append(pendingAttachments, message)
		} else {
			isStoppingPoint := message.Type == types.MessageTypeAssistant ||
				(message.Type == types.MessageTypeUser && firstBlockIsToolResult(&message))

			if isStoppingPoint && len(pendingAttachments) > 0 {
				for j := 0; j < len(pendingAttachments); j++ {
					result = append(result, pendingAttachments[j])
				}
				result = append(result, message)
				pendingAttachments = pendingAttachments[:0]
			} else {
				result = append(result, message)
			}
		}
	}
	for j := 0; j < len(pendingAttachments); j++ {
		result = append(result, pendingAttachments[j])
	}

	// reverse result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func firstBlockIsToolResult(m *types.Message) bool {
	inner, err := getInner(m)
	if err != nil || len(inner.Content) == 0 {
		return false
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) == 0 {
		return false
	}
	t, _ := blocks[0]["type"].(string)
	return t == "tool_result"
}
