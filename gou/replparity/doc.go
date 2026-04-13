// Package replparity holds no runtime logic. It documents how the Bubble Tea
// gou-demo stack mirrors claude-code/src/screens/REPL.tsx (Ink) so reviewers
// can “抄” the mapping instead of re-reading 6k lines of TS.
//
// Human-readable feature catalog (major vs secondary): docs/repl-tsx-features.md
//
// # Expectation: this is not the TS REPL shell
//
// Splitting Go across packages does not produce the same thing as REPL.tsx. gou-demo
// uses Bubble Tea + lipgloss, a smaller layout, fewer components, and a subset of
// keybindings and modals. You should not expect pixel-level or full behavioral
// reproduction of the Ink product UI here. These packages target engine-adjacent
// parity (messages, stream apply, virtual scroll, local tools, transcript subset).
// For the real shell, run the TypeScript/Ink app; closing the UX gap to Ink would be
// a separate, large effort (layout spec, theme tokens, keybinding matrix, dialogs).
//
// # Canonical TS path
//
//	claude-code/src/screens/REPL.tsx — Screen = 'prompt' | 'transcript', prompt
//	submit path, transcript modal, global keybindings, cost/spinner hooks, etc.
//
// # Screen / transcript (TS Screen, ctrl+o, frozen tail)
//
//   - gou-demo: gouDemoScreenPrompt | gouDemoScreenTranscript (cmd/gou-demo/transcript_screen.go)
//   - Enter/exit transcript, frozen snapshot: enterTranscriptScreen, exitTranscriptScreen, transcriptFrozen (*frozenTranscriptSnapshot); streaming tail length from conversation.Store.StreamingToolUses (TS streamingToolUses), filled during HTTP streaming via query.QueryDeps.OnStreamingToolUses → gou-demo Bubble Tea msg; transcript list appends frozen-prefix streaming tool rows (scroll keys gou-st-tool:*, search/export)
//   - View split: cmd/gou-demo/main.go View, listViewportH, scrollItemKeys
//   - Parity checklist: docs/plans/gou-demo-transcript-ts-parity.md
//
// # Message list / virtual scroll
//
//   - TS VirtualMessageList + useVirtualScroll → goc/gou/virtualscroll
//   - TS Messages.tsx list pipeline (filter / reorder / transcript tail cap) → goc/gou/messagesview (MessagesForScrollList, ReorderMessagesInUI)
//   - TS scrollRef (ScrollBoxHandle: scrollTo, scrollToBottom, sticky, …) → gou-demo model fields
//     (scrollTop, sticky, pendingDelta) updated in Update/keys; next View pass renders — no separate ref type.
//   - Prompt message list defaults to bubbles/viewport (go-tui-style); opt out via env in cmd/gou-demo/message_viewport_pane.go; ctrl+y fold; legacy virtualscroll via GOU_DEMO_LEGACY_VIRTUAL_MESSAGE_SCROLL=1. go-tui shares goc/gou/viewportfold for section toggles.
//   - Wheel / drag-to-scroll on the message list: cmd/gou-demo/mouse_message_list.go (tea.WithMouseCellMotion when enabled).
//     TS CLAUDE_CODE_DISABLE_MOUSE / GOU_DEMO_DISABLE_MOUSE omits SGR mouse so the host terminal can select/copy;
//     optional one-column TUI scrollbar: GOU_DEMO_MESSAGE_SCROLLBAR=1 (GOU_DEMO_NO_SCROLLBAR=1 forces off). gou-demo
//     does not use tea.WithAltScreen; redraw uses the normal terminal buffer.
//   - In-app selection (Shift+left-drag) + Ctrl+C copy + OSC 52 / pbcopy / tmux load-buffer: cmd/gou-demo/message_selection.go,
//     selection_clipboard.go (subset of TS useSelection + setClipboard).
//   - Row body: goc/gou/messagerow (SegmentsFromMessage*, tool chrome, collapsed_read_search)
//   - Render/stitch: cmd/gou-demo/main.go formatMessageSegments, renderMessageRow, measureMessageRows
//
// # Stream apply (assistant_delta, tool_use, tool_result, turn_complete)
//
//   - TS handleMessageFromStream path → goc/gou/ccbstream.Apply on conversation.Store
//   - TS useDeferredValue(messages) (yield heavy Messages during streaming): Go has no React Concurrent; gou-demo
//     instead skips full rebuildHeightCache on ccbstream assistant_delta (Apply only appends StreamingText;
//     prompt View draws streaming markdown outside virtual-scroll keys). gouStreamingToolUsesMsg skips full
//     rebuild on prompt (live tools outside scroll keys); transcript mode still rebuilds for gou-st-tool:* keys.
//     See cmd/gou-demo/stream_ui_height.go and Update ccbstream.Msg / gouStreamingToolUsesMsg branches.
//   - Optional Read/Grep/Glob tail merge: env GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL (gou/ccbstream/apply.go)
//
// # Prompt input (multiline, submit, newline keys)
//
//   - TS PromptInput → goc/gou/prompt (Bubble Tea model)
//   - gou-demo wiring: cmd/gou-demo/main.go (pr field, Update)
//
// # Query / tools (local parity, not remote ReplBridge)
//
//   - TS query engine + tool runner → goc/conversation-runtime/query, goc/ccb-engine/skilltools
//   - gou-demo turn: processuserinput.ProcessUserInput, pui.BuildDemoParams / ApplyBaseResult (cmd/gou-demo/main.go)
//   - ReplBridge / remote REPL: explicitly non-goal — docs/plans/gou-demo-repl-bridge-scope.md
//
// # Transcript pager / search / dump / editor (modal layer)
//
//   - TS ScrollKeybindingHandler, search bar, [ dump, v editor → cmd/gou-demo/transcript_search.go,
//     transcript_dump_editor.go, transcript_screen.go helpers
//
// # Loading / tool row chrome (⏺ / ⎿ / ctrl+o expand hints)
//
//   - TS MessageRow / AssistantToolUseMessage → docs/plans/gou-demo-loading-ui-parity.md,
//     cmd/gou-demo/main.go toolRowLeadPrefix / formatMessageSegments, gou/messagerow/tool_*.go
//
// # What is not ported here (use TS / other services)
//
//	useReplBridge, useRemoteSession, useSSHSession, swarm/leader permission bridges,
//	voice, frustration survey, GrowthBook, cost dialogs UI parity, IdleReturnDialog,
//	full GlobalKeybindingSetup parity — either out of scope for gou-demo or only partially mirrored.
//
// When adding a REPL.tsx behavior, extend this comment and the relevant *_parity.md
// under docs/plans/, then implement in the Go package listed above (keep files small;
// do not grow a second monolith in cmd/gou-demo/main.go if a new goc/gou/ package fits).
package replparity
