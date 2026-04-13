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
//   - Enter/exit transcript, frozen snapshot: enterTranscriptScreen, exitTranscriptScreen, transcriptFrozen (*frozenTranscriptSnapshot)
//   - View split: cmd/gou-demo/main.go View, listViewportH, scrollItemKeys
//   - Parity checklist: docs/plans/gou-demo-transcript-ts-parity.md
//
// # Message list / virtual scroll
//
//   - TS VirtualMessageList + useVirtualScroll → goc/gou/virtualscroll
//   - Row body: goc/gou/messagerow (SegmentsFromMessage*, tool chrome, collapsed_read_search)
//   - Render/stitch: cmd/gou-demo/main.go formatMessageSegments, renderMessageRow, measureMessageRows
//
// # Stream apply (assistant_delta, tool_use, tool_result, turn_complete)
//
//   - TS handleMessageFromStream path → goc/gou/ccbstream.Apply on conversation.Store
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
