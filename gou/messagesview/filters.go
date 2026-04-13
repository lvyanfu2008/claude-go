package messagesview

import (
	"encoding/json"

	"goc/types"
)

// MaxTranscriptMessagesWithoutVirtualScroll mirrors TS
// MAX_MESSAGES_TO_SHOW_IN_TRANSCRIPT_MODE when virtual scroll is off
// (Messages.tsx shouldTruncate).
const MaxTranscriptMessagesWithoutVirtualScroll = 30

// nullRenderingAttachmentTypes matches claude-code/src/components/messages/nullRenderingAttachments.ts
// NULL_RENDERING_TYPES (subset used for list-budget filtering before render).
var nullRenderingAttachmentTypes = map[string]struct{}{
	"hook_success":             {},
	"hook_additional_context":  {},
	"hook_cancelled":           {},
	"command_permissions":      {},
	"agent_mention":            {},
	"budget_usd":               {},
	"critical_system_reminder": {},
	"edited_image_file":        {},
	"edited_text_file":         {},
	"opened_file_in_ide":       {},
	"output_style":             {},
	"plan_mode":                {},
	"plan_mode_exit":           {},
	"plan_mode_reentry":        {},
	"structured_output":        {},
	"team_context":             {},
	"todo_reminder":            {},
	"context_efficiency":       {},
	"deferred_tools_delta":     {},
	"mcp_instructions_delta":   {},
	"companion_intro":          {},
	"token_usage":              {},
	"ultrathink_effort":        {},
	"max_turns_reached":        {},
	"task_reminder":            {},
	"auto_mode":                {},
	"auto_mode_exit":           {},
	"output_token_usage":       {},
	"verify_plan_reminder":     {},
	"current_session_memory":   {},
	"compaction_reminder":      {},
	"date_change":              {},
}

// IsNullRenderingAttachment ports isNullRenderingAttachment from nullRenderingAttachments.ts.
func IsNullRenderingAttachment(msg types.Message) bool {
	if msg.Type != types.MessageTypeAttachment || len(msg.Attachment) == 0 {
		return false
	}
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(msg.Attachment, &head) != nil {
		return false
	}
	_, ok := nullRenderingAttachmentTypes[head.Type]
	return ok
}

// ShouldShowUserMessage ports shouldShowUserMessage from messages.ts (Kairos channel branch omitted).
func ShouldShowUserMessage(msg types.Message, transcriptMode bool) bool {
	if msg.Type != types.MessageTypeUser {
		return true
	}
	if msg.IsMeta != nil && *msg.IsMeta {
		return false
	}
	if msg.IsVisibleInTranscriptOnly != nil && *msg.IsVisibleInTranscriptOnly && !transcriptMode {
		return false
	}
	return true
}

// DropProgress removes type "progress" rows (Messages.tsx filters before reorder).
func DropProgress(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]types.Message, 0, len(messages))
	for i := range messages {
		if messages[i].Type == types.MessageTypeProgress {
			continue
		}
		out = append(out, messages[i])
	}
	return out
}

// DropNullRenderingAttachments removes attachment rows that TS AttachmentMessage renders as null.
func DropNullRenderingAttachments(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]types.Message, 0, len(messages))
	for i := range messages {
		if IsNullRenderingAttachment(messages[i]) {
			continue
		}
		out = append(out, messages[i])
	}
	return out
}

// FilterShouldShowUserMessage keeps user rows according to transcript vs prompt semantics.
func FilterShouldShowUserMessage(messages []types.Message, transcriptMode bool) []types.Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]types.Message, 0, len(messages))
	for i := range messages {
		if ShouldShowUserMessage(messages[i], transcriptMode) {
			out = append(out, messages[i])
		}
	}
	return out
}

// maybeTranscriptTail mirrors Messages.tsx slice(-MAX_MESSAGES_TO_SHOW_IN_TRANSCRIPT_MODE)
// when shouldTruncate; returns a new slice only when trimming applies.
func maybeTranscriptTail(messages []types.Message, transcriptMode, showAllInTranscript, virtualScrollEnabled bool) []types.Message {
	if !transcriptMode || showAllInTranscript || virtualScrollEnabled {
		return messages
	}
	capN := MaxTranscriptMessagesWithoutVirtualScroll
	if len(messages) <= capN {
		return messages
	}
	return slicesCloneLastN(messages, capN)
}

func slicesCloneLastN(messages []types.Message, n int) []types.Message {
	if n < 1 || len(messages) <= n {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	start := len(messages) - n
	out := make([]types.Message, n)
	copy(out, messages[start:])
	return out
}
