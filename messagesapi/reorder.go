package messagesapi

import (
	"goc/types"
)

// reorderAttachmentsForAPI mirrors src/utils/messages.ts reorderAttachmentsForAPI.
//
// Trailing attachments after the last user must not be flushed at the next assistant when scanning
// backward (e.g. […, assistant, user, attachment] should keep attachment after that user). We flush
// pending attachments immediately before a plain user when that user is directly followed by an
// attachment in the original slice and either there is no assistant earlier than that user or the
// transcript has at least two user rows (multi-turn); otherwise pending continues to bubble to the
// stopping assistant as before.
func reorderAttachmentsForAPI(messages []types.Message) []types.Message {
	var result []types.Message
	var pendingAttachments []types.Message
	userCount := countUserMessagesIn(messages)

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
				if message.Type == types.MessageTypeUser && !isStoppingPoint &&
					len(pendingAttachments) > 0 &&
					i+1 < len(messages) && messages[i+1].Type == types.MessageTypeAttachment &&
					(!hasAssistantBefore(messages, i) || userCount >= 2) {
					for j := 0; j < len(pendingAttachments); j++ {
						result = append(result, pendingAttachments[j])
					}
					pendingAttachments = pendingAttachments[:0]
				}
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

func countUserMessagesIn(messages []types.Message) int {
	n := 0
	for _, m := range messages {
		if m.Type == types.MessageTypeUser {
			n++
		}
	}
	return n
}

func hasAssistantBefore(messages []types.Message, i int) bool {
	for j := 0; j < i; j++ {
		if messages[j].Type == types.MessageTypeAssistant {
			return true
		}
	}
	return false
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
