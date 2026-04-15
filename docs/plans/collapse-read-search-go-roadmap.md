# `collapsed_read_search` — Go parity roadmap

Source of truth: `claude-code/src/utils/collapseReadSearch.ts` + `CollapsedReadSearchContent.tsx`.

## Done (this iteration)

- **`IsSearchOrReadBashCommand`** (`gou/messagerow/bash_search_read.go`): mirrors `BashTool.tsx` `isSearchOrReadBashCommand` (same command sets; pipeline / `&&` / `||` via `mvdan.cc/sh/v3/syntax`). Continuation-line join matches TS `splitCommandWithOperators` backslash-newline rule.
- **`CollapseReadSearchTail`**: trailing rollup now includes **Bash** / **BashZog** when the command classifies as search, read, or list (same rule as TS without fullscreen-only “all bash” bucket). Counts: **list** branch before **search** before **read ops**, aligned with `collapseReadSearchGroups` ordering. **`ListCount`** set on the synthetic `collapsed_read_search` message when applicable.
- **Tests**: `bash_search_read_test.go`, updated `collapse_roll_up_test.go` (non-collapsible bash uses `git status`; `ls` + Read merges into one row).

## Still out of scope (TS-only today)

- Fullscreen **`isBash`** “Ran N bash commands” without search/read/list (`isFullscreenEnvEnabled` + generic Bash).
- Memory / team memory / MCP / Snip / ToolSearch / hooks / `relevant_memories` attachment absorption.
- Full `collapseReadSearchGroups` over the **entire** message list (Go keeps **tail-only** rollup behind `GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1`).
- Verbose / transcript line-by-line rendering parity with `CollapsedReadSearchContent` (`VerboseToolUse`).

## Suggested next steps

1. Optional env **`GOU_DEMO_COLLAPSE_ALL_BASH=1`**: map TS fullscreen behavior (every Bash pair rolls up into `bashCount`).
2. Golden tests: share fixture strings with TS for `isSearchOrReadBashCommand` edge cases.
3. Wire **full** `collapseReadSearchGroups` into the message pipeline if gou-demo should match Ink ordering beyond the tail.
