# `collapsed_read_search` — Go parity roadmap

Source of truth: `claude-code/src/utils/collapseReadSearch.ts` + `CollapsedReadSearchContent.tsx`.

## Done

- **`IsSearchOrReadBashCommand`** (`gou/messagerow/bash_search_read.go`): mirrors `BashTool.tsx` `isSearchOrReadBashCommand` (same command sets; pipeline / `&&` / `||` via `mvdan.cc/sh/v3/syntax`). Continuation-line join matches TS `splitCommandWithOperators` backslash-newline rule.
- **`CollapseReadSearchTail`**: store-side tail merge when **`GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1`** (`gou/ccbstream/apply.go`).
- **`CollapseReadSearchGroupsInList`**: display pipeline full-list merge (`gou/messagesview/pipeline.go` after `ApplyGrouping`; always on). Do not combine with tail merge on the same store unless you understand double-collapse risk.
- **`GOU_DEMO_COLLAPSE_ALL_BASH=1`**: `CollapseAllBashFromEnv` — any Bash in rollup + **`bashCount`** for generic commands.
- **`SearchReadSummaryText`**: bash phrases aligned with `CollapsedReadSearchContent.tsx`.
- **`RenderOpts.VerboseCollapsedReadSearch`**: transcript screen renders nested `msg.Messages` under collapsed rows (TS `verbose || isTranscriptMode` analogue); **`ShowAllInTranscript`** still applies when ctrl+e show-all is on (verbose branch takes precedence for collapsed body).
- **Doc**: `claude-code/docs/features/tool-use-grouping-and-transcript-ui.md` — Layer C / Go subsection (monorepo path).

## Still out of scope (TS-only)

- TS **`isFullscreenEnvEnabled()`** auto-detection; Go uses explicit env vars.
- Memory / team / MCP / Snip / ToolSearch / hooks / `relevant_memories` in **`collapseReadSearchGroups`** (TS `createCollapsedGroup` full accumulator).
- Ink **`VerboseToolUse`** per-tool schema / resolve styling parity (Go uses `SegmentsFromMessage` on nested rows).

## Optional next steps

1. Port remaining **`collapseReadSearchGroups`** branches (attachments, hooks, git scan) when needed.
2. Map **`GOU_DEMO_COLLAPSE_ALL_BASH`** to TS fullscreen env for one-knob parity.
