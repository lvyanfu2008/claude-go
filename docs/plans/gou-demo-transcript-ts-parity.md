# gou-demo transcript mode vs TS `REPL.tsx`

Reference: [`claude-code/src/screens/REPL.tsx`](../../../claude-code/src/screens/REPL.tsx) (`Screen = 'prompt' | 'transcript'`), [`claude-code/src/hooks/useGlobalKeybindings.tsx`](../../../claude-code/src/hooks/useGlobalKeybindings.tsx).

## Behavior mapping

| TS | gou-demo (this port) | Notes |
|----|----------------------|--------|
| `ctrl+o` → `app:toggleTranscript` | `ctrl+o` toggles `prompt` ↔ `transcript` | Matches default keybinding |
| Enter transcript → `onEnterTranscript` freezes lengths | On enter, `transcriptFreezeN = len(Messages)` | TS stores `messagesLength` + `streamingToolUsesLength`; Go uses message count only (no separate streaming tool-use map) |
| Exit transcript: `transcript:exit` on `Esc` / `ctrl+c` when search bar closed | `Esc` / `ctrl+c` / `q` exit transcript when **search bar is closed** | In **prompt** mode, `q` / `Esc` / `ctrl+c` still **quit** the program (unchanged) |
| Transcript `q` (less-style) | Same as above when `screen == transcript` | TS `useInput` bare `q` calls `handleExitTranscript` |
| `ctrl+e` → `transcript:toggleShowAll` | `ctrl+e` toggles `transcriptShowAll` | Drives [messagerow.RenderOpts.ShowAllInTranscript](../../gou/messagerow/segment.go): expands `collapsed_read_search` (files + search terms) and inlines `grouped_tool_use` nested messages/results |
| Scroll `g`/`G`/`j`/`k`, `ctrl+u`/`ctrl+d`, `ctrl+b`/`ctrl+f`, bare `b` | **Ported** (subset of TS `modalPagerAction` in `ScrollKeybindingHandler.tsx`) | When **search bar closed**; **not** bare `space` (page down in less) to avoid accidental scroll |
| Search `/`, bar `Esc` / `Enter`, `n`/`N`, resize clears | **`/`** opens search bar; **`Esc`** in bar clears search state (stay in transcript); **`Enter`** closes bar but keeps query for **`n`/`N`**; **`n`/`N`** step matches when bar closed and query non-empty; **column change** clears search | Plain-text substring match over frozen messages (no TS highlight overlay) |
| `[` dump mode, `v` external editor | **`[`** sets dump + show-all, exits alt-screen when enabled, **`tea.Printf`** plain export to scrollback; **`v`** writes frozen transcript to temp (width `max(80, cols−6)`), strips trailing line spaces, **`tea.ExecProcess`** `$VISUAL`/`$EDITOR` (status + 4s clear like TS) | Go uses Bubble Tea scrollback + exec; TS Ink unwrap + `renderMessagesToPlainText` |
| New model events while in transcript | **No auto-scroll** to tail | TS frozen slice ignores new tail until exit |

## Acceptance (manual)

1. Start `go run ./cmd/gou-demo` from `goc` with seed messages; press **ctrl+o** → header/footer show transcript mode; list shows messages up to the freeze point.
2. **Scroll up** with arrows; press **ctrl+o** again → returns to prompt with **restored** scroll position from before step 1.
3. In transcript, **Esc**, **ctrl+c**, and **q** each return to prompt **without** exiting the app (when search bar is not open).
4. In transcript, **ctrl+e** toggles expanded rows for collapsed/grouped messages (and footer shows expand on/off).
5. In transcript, press **`/`**, type a substring (e.g. `seed`), confirm status shows matches and **`n`/`N`** jumps between hits; **`Esc`** in the bar clears search; resize terminal clears search.
6. **`[`** (search bar closed): footer switches to dump hint; with default alt-screen, plain transcript prints to **scrollback**; **`/`** / **`n`/`N`** and **pager keys** are inactive until exit transcript (TS `!dumpMode` / no `ScrollKeybindingHandler`). **`v`** writes temp file and runs **`$VISUAL`/`$EDITOR`** (blocking `tea.ExecProcess`); empty env shows **wrote … · no $VISUAL/$EDITOR set**.
7. Trigger a **streaming** turn (fake stream or real query): while in transcript, the pane must **not** jump to new assistant chunks; exit transcript to see live tail.

## Automated

- `go test ./cmd/gou-demo/...` — transcript helpers, search plain-text, **REPL** `RunToolUseChan` + `ParityToolRunner` integration.
- `go test ./gou/messagerow/...` — `RenderOpts.ShowAllInTranscript` segment tests.

## Related

- REPL tool permissions vs TS: [go-repl-permissions-parity.md](./go-repl-permissions-parity.md).
- ReplBridge non-goal: [gou-demo-repl-bridge-scope.md](./gou-demo-repl-bridge-scope.md).

## Phase 3 (optional)

| TS transcript | Status |
|-----------------|--------|
| `[` dump to scrollback + expand all | **Ported** (`transcriptDumpMode`, `tea.ExitAltScreen` + `tea.Printf` when alt-screen on) |
| `v` open full transcript in `$VISUAL` / `$EDITOR` | **Ported** (`transcript_dump_editor.go`; double-tap guarded while busy) |
| Modal pager bare `space` (full page down), `ctrl+n` / `ctrl+p` line scroll | Not implemented (space omitted intentionally) |
