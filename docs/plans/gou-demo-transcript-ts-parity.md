# gou-demo transcript mode vs TS `REPL.tsx`

Reference: [`claude-code/src/screens/REPL.tsx`](../../../claude-code/src/screens/REPL.tsx) (`Screen = 'prompt' | 'transcript'`), [`claude-code/src/hooks/useGlobalKeybindings.tsx`](../../../claude-code/src/hooks/useGlobalKeybindings.tsx).

## Behavior mapping

| TS | gou-demo (this port) | Notes |
|----|----------------------|--------|
| `ctrl+o` → `app:toggleTranscript` | `ctrl+o` toggles `prompt` ↔ `transcript` | Matches default keybinding |
| Enter transcript → `onEnterTranscript` freezes lengths | On enter, `transcriptFreezeN = len(Messages)` | TS stores `messagesLength` + `streamingToolUsesLength`; Go uses message count only (no separate streaming tool-use map) |
| Exit transcript: `transcript:exit` on `Esc` / `ctrl+c` when search bar closed | `Esc` / `ctrl+c` / `q` exit transcript (return to prompt) | In **prompt** mode, `q` / `Esc` / `ctrl+c` still **quit** the program (unchanged) |
| Transcript `q` (less-style) | Same as above when `screen == transcript` | TS `useInput` bare `q` calls `handleExitTranscript` |
| `ctrl+e` → `transcript:toggleShowAll` (when `!virtualScrollActive`) | `ctrl+e` toggles `transcriptShowAll` | TS uses this for legacy non-virtual transcript cap; Go keeps flag for future collapsed expansion / footer label |
| Scroll `g`/`G`/`j`/`k`/… | **Not ported** (ScrollKeybindingHandler) | gou-demo keeps **↑↓ PgUp PgDn End** in transcript |
| Search `/`, `n`/`N` | **Not ported** | Requires jump/highlight stack |
| `[` dump mode, `v` external editor | **Not ported** | Optional later milestone |
| New model events while in transcript | **No auto-scroll** to tail | TS frozen slice ignores new tail until exit |

## Acceptance (manual)

1. Start `go run ./cmd/gou-demo` from `goc` with seed messages; press **ctrl+o** → header/footer show transcript mode; list shows messages up to the freeze point.
2. **Scroll up** with arrows; press **ctrl+o** again → returns to prompt with **restored** scroll position from before step 1.
3. In transcript, **Esc**, **ctrl+c**, and **q** each return to prompt **without** exiting the app.
4. In transcript, **ctrl+e** toggles footer “show all” on/off (no crash).
5. Trigger a **streaming** turn (fake stream or real query): while in transcript, the pane must **not** jump to new assistant chunks; exit transcript to see live tail.

## Automated

- `go test ./cmd/gou-demo/...` covers `transcriptScreen` helpers (freeze clamp, key routing).
