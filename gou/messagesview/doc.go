// Package messagesview implements the message-list data pipeline that
// claude-code/src/components/Messages.tsx builds before VirtualMessageList
// (normalize / filter / reorder / optional transcript cap). It does not render
// rows, stream markdown, or own Bubble Tea state — only pure []types.Message
// transforms shared by gou-demo and tests.
//
// TS mapping (MessagesImpl useMemo chain, simplified):
//
//   - filter type !== 'progress'              → DropProgress
//   - filter !isNullRenderingAttachment       → DropNullRenderingAttachments
//   - filter shouldShowUserMessage            → ShouldShowUserMessage
//   - reorderMessagesInUI(…)                  → ReorderMessagesInUI
//   - transcript && !showAll && !virtual gate → last N cap (see MaxTranscriptMessagesWithoutVirtualScroll)
//   - applyGrouping(…)                        → ApplyGrouping
//
// Not ported here (still in stream/store/messagerow or future work):
// normalizeMessages splitting, getMessagesAfterCompactBoundary,
// collapse* chains, brief-only filters, computeSliceStart for non-virtual
// fullscreen cap, buildMessageLookups.
package messagesview
