package messagesview

import (
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
