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
	work = maybeTranscriptTail(work, o.TranscriptMode, o.ShowAllInTranscript, o.VirtualScrollEnabled)
	work = ApplyGrouping(work, o.Verbose)
	work = messagerow.CollapseReadSearchGroupsInList(work)
	return work
}
