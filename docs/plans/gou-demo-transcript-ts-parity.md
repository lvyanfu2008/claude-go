# gou-demo transcript mode vs TS `REPL.tsx`

Reference: [`claude-code/src/screens/REPL.tsx`](../../../claude-code/src/screens/REPL.tsx) (`Screen = 'prompt' | 'transcript'`), [`claude-code/src/hooks/useGlobalKeybindings.tsx`](../../../claude-code/src/hooks/useGlobalKeybindings.tsx).

## Behavior mapping

| TS | gou-demo (this port) | Notes |
|----|----------------------|--------|
| `ctrl+o` → `app:toggleTranscript` | `ctrl+o` toggles `prompt` ↔ `transcript` | Matches default keybinding |
| Enter transcript → `onEnterTranscript` freezes lengths | On enter, `transcriptFrozen = &frozenTranscriptSnapshot{MessagesLen, StreamingToolUsesLen}` (`transcript_screen.go`) | Matches TS `{ messagesLength, streamingToolUsesLength }`. `StreamingToolUsesLen` is `len(store.StreamingToolUses)` ([`gou/conversation`](../../gou/conversation/store.go)). **HTTP 流式**：[`query` streaming 循环](../../conversation-runtime/query/streaming_loop.go) 在 `content_block_*` 后调用 `QueryDeps.OnStreamingToolUses`，`gou-demo` 写入 store；`message_stop` 传 `nil` 清空（见 `TestStreamingParity_OnStreamingToolUsesSnapshots`）。ccb NDJSON 仍无增量 tool 事件，故多为空。 |
| Exit transcript: `transcript:exit` on `Esc` / `ctrl+c` when search bar closed | `Esc` / `ctrl+c` / `q` exit transcript when **search bar is closed** | In **prompt** mode, `q` / `Esc` / `ctrl+c` still **quit** the program (unchanged) |
| Transcript `q` (less-style) | Same as above when `screen == transcript` | TS `useInput` bare `q` calls `handleExitTranscript` |
| `ctrl+e` → `transcript:toggleShowAll` | `ctrl+e` toggles `transcriptShowAll` | Drives [messagerow.RenderOpts.ShowAllInTranscript](../../gou/messagerow/segment.go): expands `collapsed_read_search` (files + search terms) and inlines `grouped_tool_use` nested messages/results |
| Scroll **`home`** / **`end`** and **`ctrl+home`** / **`ctrl+end`** (TS `scroll:top` / `scroll:bottom` in [`defaultBindings.ts`](../../../claude-code/src/keybindings/defaultBindings.ts)), `g`/`G`/`j`/`k`, `ctrl+u`/`ctrl+d`, `ctrl+b`/`ctrl+f`, bare `b`, bare **`space`** (full page down), **`ctrl+n`** / **`ctrl+p`** (line down/up) | **Ported** (`modalPagerAction`: bare `home`/`end`; ctrl pair via separate keybinding route in TS, same scroll targets) | When **search bar closed** (`isModal={!searchOpen}`); inactive in **dump** mode |
| Search `/`, bar `Esc` / `Enter`, `n`/`N`, resize clears | **`/`** opens search bar; **`Esc`** in bar clears search state (stay in transcript); **`Enter`** closes bar but keeps query for **`n`/`N`**; **`n`/`N`** step matches when bar closed and query non-empty; **column change** clears search | Plain-text substring match over **`messagesForScroll()`** (same order as TS `reorderMessagesInUI`) plus streaming tool rows; visible plain segments get **`lipgloss`** highlight (`highlightSearchPlain` / `transcriptSearchHLStyle`, dump mode off) |
| `reorderMessagesInUI` (tool_use / tool_result / Pre–Post hook grouping) | **`ReorderMessagesInUI`** in [`gou/messagesview/reorder_ui.go`](../../gou/messagesview/reorder_ui.go) as part of **`MessagesForScrollList`**; virtual scroll keys, **`View`**, height cache, search haystack, and **`[`** / **`v`** plain export all use **`messagesForScroll()`** | Synthetic `transcriptStreamingToolUses` rows still append after frozen messages (TS) |
| `[` dump mode, `v` external editor | **`[`** sets dump + show-all, **`tea.Printf`** plain export to scrollback; **`v`** writes frozen transcript to temp (width `max(80, cols−6)`), strips trailing line spaces, **`tea.ExecProcess`** `$VISUAL`/`$EDITOR` (status + 4s clear like TS) | Go uses Bubble Tea `Printf` + exec; TS Ink unwrap + `renderMessagesToPlainText` |
| New model events while in transcript | **No auto-scroll** to tail | TS frozen slice ignores new tail until exit |
| `transcriptStreamingToolUses` synthetic rows in list | **Ported**: virtual-scroll keys append `gou-st-tool:*` after frozen messages; `View` / height cache / `[`/`v` plain export include `transcriptStreamingToolsForView()` slice (`slice(0, frozen.StreamingToolUsesLength)` of live store list); search includes tool name/id/input |

## Acceptance (manual)

1. Start `go run ./cmd/gou-demo` from `goc` with seed messages; press **ctrl+o** → header/footer show transcript mode; list shows messages up to the freeze point.
2. **Scroll up** with arrows; press **ctrl+o** again → returns to prompt with **restored** scroll position from before step 1.
3. In transcript, **Esc**, **ctrl+c**, and **q** each return to prompt **without** exiting the app (when search bar is not open).
4. In transcript, **ctrl+e** toggles expanded rows for collapsed/grouped messages (and footer shows expand on/off).
5. In transcript, press **`/`**, type a substring (e.g. `seed`), confirm status shows matches and **`n`/`N`** jumps between hits; while **not** in **`[`** dump mode, matching substrings in the message pane should show **search highlight** (lipgloss background on hits, same intent as TS `useSearchHighlight`). **`Esc`** in the bar clears search; resize terminal clears search.
6. With search bar **closed**, **home** / **end** and **ctrl+home** / **ctrl+end** jump to top / bottom, **space** page-downs one viewport, **ctrl+n** / **ctrl+p** move one line (TS `modalPagerAction` + `scroll:top`/`scroll:bottom`). With search bar **open**, **arrows** and those pager keys do **not** scroll (TS `isModal={!searchOpen}`).
7. **`[`** (search bar closed): footer switches to dump hint; plain transcript prints to **scrollback** via **`tea.Printf`**; **`/`** / **`n`/`N`** and **pager keys** are inactive until exit transcript (TS `!dumpMode` / no `ScrollKeybindingHandler`). **`v`** writes temp file and runs **`$VISUAL`/`$EDITOR`** (blocking `tea.ExecProcess`); empty env shows **wrote … · no $VISUAL/$EDITOR set**.
8. Trigger a **streaming** turn (fake stream or real query): while in transcript, the pane must **not** jump to new assistant chunks; exit transcript to see live tail.

## Automated

- `go test ./cmd/gou-demo/...` — transcript helpers, search plain-text, **REPL** `RunToolUseChan` + `ParityToolRunner` integration.
- `go test ./gou/messagerow/...` — `RenderOpts.ShowAllInTranscript` segment tests.

Transcript-focused cases in `cmd/gou-demo` (non-exhaustive): pager (**space**, **ctrl+n**/**ctrl+p**, **home**/**end**, **ctrl+home**/**ctrl+end**), search bar swallowing pager keys, **`[`** dump mode + show-all + dump `tea.Cmd`, **`v`** temp export + `handleTranscriptEditorChainMsg` when `$VISUAL`/`$EDITOR` unset, double-**`v`** while editor prep is in flight, `exitTranscriptScreenWithPostCmd` after dump, `transcript_dump_editor_test.go` (export width, bracket scrollback helpers), **`gou/messagesview` tests** (tool group ordering and list pipeline).

## Related

- Loading / spinner / tool activity vs TS: [gou-demo-loading-ui-parity.md](./gou-demo-loading-ui-parity.md).
- REPL tool permissions vs TS: [go-repl-permissions-parity.md](./go-repl-permissions-parity.md).
- ReplBridge non-goal: [gou-demo-repl-bridge-scope.md](./gou-demo-repl-bridge-scope.md).

### Deferred optional (after transcript milestone)

Smaller next steps are usually **loading / tool-row chrome** ([gou-demo-loading-ui-parity.md](./gou-demo-loading-ui-parity.md): spinner tips, `GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL`, etc.). **Per-inner-call REPL permissions** ([go-repl-permissions-parity.md](./go-repl-permissions-parity.md) Future work) is a larger, separate change—only schedule it if you need stricter alignment with TS `permissions.ts` for nested tools.

## Phase 3 (optional)

| TS transcript | Status |
|-----------------|--------|
| `[` dump to scrollback + expand all | **Ported** (`transcriptDumpMode`, `tea.Printf` to scrollback) |
| `v` open full transcript in `$VISUAL` / `$EDITOR` | **Ported** (`transcript_dump_editor.go`; double-tap guarded while busy) |
| Modal pager bare `space` (full page down), `ctrl+n` / `ctrl+p` line scroll, **`home` / `end`** and **`ctrl+home` / `ctrl+end`** top/bottom | **Ported** |
