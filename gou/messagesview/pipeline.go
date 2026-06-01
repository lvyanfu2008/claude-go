package messagesview

import (
	"encoding/json"
	"fmt"

	"goc/gou/messagerow"
	"goc/types"
)

// ScrollListOpts configures MessagesForScrollList (TS Messages.tsx list inputs).
type ScrollListOpts struct {
	// TranscriptMode is true on the ctrl+o transcript screen ('transcript' Screen).
	TranscriptMode bool
	// ShowAllInTranscript is TS showAllInTranscript (or dump/editor modes that need full history).
	ShowAllInTranscript bool
	// VirtualScrollEnabled is TS virtualScrollRuntimeGate (scrollRef present && !CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL).
	VirtualScrollEnabled bool
	// Verbose is TS verbose mode (skips grouping, renders tools as single blocks).
	Verbose bool
	// ResolvedToolUseIDs is the set of tool_use_ids that have resolved results (non-nil for collapse).
	ResolvedToolUseIDs map[string]struct{}
}

// splitMixedMessages splits assistant messages that mix text and tool_use content blocks
// into separate messages so that CollapseReadSearchGroupsInList can correctly collapse
// consecutive tool_use blocks. Also splits user messages with multiple tool_result blocks.
// This is needed when the model produces [text, tool_use, tool_use] in a single message (non-thinking mode).
func splitMixedMessages(messages []types.Message) []types.Message {
	var result []types.Message
	for _, msg := range messages {
		blocks := messagerow.MessageContentBlocks(msg)
		if len(blocks) <= 1 {
			result = append(result, msg)
			continue
		}
		switch msg.Type {
		case types.MessageTypeAssistant:
			result = append(result, splitAssistantBlocks(msg, blocks)...)
		case types.MessageTypeUser:
			result = append(result, splitUserBlocks(msg, blocks)...)
		default:
			result = append(result, msg)
		}
	}
	return result
}

func splitAssistantBlocks(msg types.Message, blocks []types.MessageContentBlock) []types.Message {
	var result []types.Message
	var textBlocks []types.MessageContentBlock
	splitIdx := 0

	flushText := func() {
		if len(textBlocks) == 0 {
			return
		}
		clone := cloneMessage(msg)
		clone.UUID = fmt.Sprintf("%s-split-%d", msg.UUID, splitIdx)
		splitIdx++
		raw, _ := json.Marshal(textBlocks)
		clone.Content = raw
		result = append(result, clone)
		textBlocks = nil
	}

	for _, b := range blocks {
		if b.Type == "text" {
			textBlocks = append(textBlocks, b)
		} else if b.Type == "tool_use" {
			flushText()
			clone := cloneMessage(msg)
			clone.UUID = fmt.Sprintf("%s-split-%d", msg.UUID, splitIdx)
			splitIdx++
			raw, _ := json.Marshal([]types.MessageContentBlock{b})
			clone.Content = raw
			result = append(result, clone)
		}
	}
	flushText()
	if len(result) == 0 {
		result = append(result, msg)
	}
	return result
}

func splitUserBlocks(msg types.Message, blocks []types.MessageContentBlock) []types.Message {
	toolResultCount := 0
	for _, b := range blocks {
		if b.Type == "tool_result" {
			toolResultCount++
		}
	}
	if toolResultCount <= 1 {
		return []types.Message{msg}
	}
	var result []types.Message
	splitIdx := 0
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		clone := cloneMessage(msg)
		clone.UUID = fmt.Sprintf("%s-split-%d", msg.UUID, splitIdx)
		splitIdx++
		raw, _ := json.Marshal([]types.MessageContentBlock{b})
		clone.Content = raw
		result = append(result, clone)
	}
	if len(result) == 0 {
		result = append(result, msg)
	}
	return result
}

func cloneMessage(msg types.Message) types.Message {
	clone := msg
	clone.Messages = nil
	clone.Results = nil
	clone.DisplayMessage = nil
	return clone
}

// MessagesForScrollList returns UI-ordered messages for virtual scroll, search haystack,
// and plain export — the slice VirtualMessageList would receive after Messages.tsx pre-reorder filters.
// Caller supplies a defensive clone if the underlying store must not be mutated (ReorderMessagesInUI does not mutate).
func MessagesForScrollList(messages []types.Message, o ScrollListOpts) []types.Message {
	if len(messages) == 0 {
		return nil
	}
	work := DropProgress(messages)
	work = DropNullRenderingAttachments(work)
	work = FilterShouldShowUserMessage(work, o.TranscriptMode)
	work = splitMixedMessages(work)
	work = ReorderMessagesInUI(work)
	work = ApplyGrouping(work, o.Verbose)
	// Collapse consecutive Read/Grep/Glob tool_use + tool_result pairs into summary groups
	if !o.TranscriptMode {
		work = messagerow.CollapseReadSearchGroupsInList(work, o.ResolvedToolUseIDs)
	}
	// Apply transcript tail after grouping to ensure correct message count
	work = maybeTranscriptTail(work, o.TranscriptMode, o.ShowAllInTranscript, o.VirtualScrollEnabled)
	return work
}
