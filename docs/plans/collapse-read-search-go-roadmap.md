# `collapsed_read_search` — Go parity roadmap

Source of truth: `claude-code/src/utils/collapseReadSearch.ts` + `CollapsedReadSearchContent.tsx`.

## Done (this iteration)

- **`IsSearchOrReadBashCommand`** (`gou/messagerow/bash_search_read.go`): mirrors `BashTool.tsx` `isSearchOrReadBashCommand` (same command sets; pipeline / `&&` / `||` via `mvdan.cc/sh/v3/syntax`). Continuation-line join matches TS `splitCommandWithOperators` backslash-newline rule.
- **`CollapseReadSearchTail`**: trailing rollup includes **Bash** / **BashZog** when TS-classified search/read/list; optional **`GOU_DEMO_COLLAPSE_ALL_BASH=1`** (`CollapseAllBashFromEnv`) rolls up **any** Bash pair and increments **`bashCount`** for commands that are not search/read/list (TS fullscreen `isBash` bucket). Counts: **list** before **search** before **read ops** before **generic bash**. **`ListCount`** / **`BashCount`** on the synthetic message when applicable.
- **`SearchReadSummaryText`**: includes bash phrases aligned with `CollapsedReadSearchContent.tsx` (“Running/running … bash commands”, “Ran/ran …”).
- **Tests**: `bash_search_read_test.go` (incl. redirect fixtures), `collapse_roll_up_test.go` (incl. `collapseAllBash` + summary), `summary_test.go` (bash-only summary).

## Still out of scope (TS-only today)

- TS **`isFullscreenEnvEnabled()`** auto-detection (`CLAUDE_CODE_NO_FLICKER`, `USER_TYPE`, tmux); Go uses explicit **`GOU_DEMO_COLLAPSE_ALL_BASH`** instead.
- Memory / team memory / MCP / Snip / ToolSearch / hooks / `relevant_memories` attachment absorption.
- Full `collapseReadSearchGroups` over the **entire** message list (Go keeps **tail-only** rollup behind `GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1`).
- Verbose / transcript line-by-line rendering parity with `CollapsedReadSearchContent` (`VerboseToolUse`).

## Suggested next steps (Phase 3 — pick separately)

1. **Full `collapseReadSearchGroups` port** — only if gou-demo must match Ink **main list** ordering beyond the tail; large change.
2. **Verbose collapsed rows** — per-tool lines in transcript when `ShowAllInTranscript` / parity with `VerboseToolUse`; separate PR.
3. Optional: map **`GOU_DEMO_COLLAPSE_ALL_BASH`** to TS **`CLAUDE_CODE_NO_FLICKER`** / **`USER_TYPE`** when product wants default fullscreen without a second env.
