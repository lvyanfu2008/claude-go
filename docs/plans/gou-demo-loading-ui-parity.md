# gou-demo loading UI vs TS Ink

Reference: [`claude-code/src/components/Spinner.tsx`](../../../claude-code/src/components/Spinner.tsx), [`spinnerVerbs.ts`](../../../claude-code/src/constants/spinnerVerbs.ts), [`tipRegistry.ts`](../../../claude-code/src/services/tips/tipRegistry.ts), tool `getActivityDescription` / `getToolUseSummary` under [`claude-code/src/tools/`](../../../claude-code/src/tools/), [`CtrlOToExpand.tsx`](../../../claude-code/src/components/CtrlOToExpand.tsx).

## Behavior mapping

| TS | gou-demo | Notes |
|----|----------|-------|
| Tool row activity + summary | [`messagerow/tool_activity.go`](../../gou/messagerow/tool_activity.go) + [`segment.go`](../../gou/messagerow/segment.go) | Read path uses `filepath.Rel` to cwd when safe; otherwise cleaned path (TS `getDisplayPath` may differ slightly). |
| `attachment` rows (e.g. `skill_listing`) | [`messagerow/attachment_segment.go`](../../gou/messagerow/attachment_segment.go) | Matches TS `AttachmentMessage`: no `attachment` role header; `skill_listing` shows **`N skill(s) available`** with bold count, hides when `isInitial` (store JSON includes `skillCount` / `isInitial` from [`AppendSkillListingForAPI`](../../commands/skill_listing_delta.go)). |
| Tool rows (TS `AssistantToolUseMessage` + ⎿) | [`messagerow/tool_chrome.go`](../../gou/messagerow/tool_chrome.go), [`cmd/gou-demo/main.go`](../../cmd/gou-demo/main.go) `formatMessageSegments` | Row1: **figures.BLACK_CIRCLE** + bold user-facing name + TS-style `renderToolUseMessage` paren text (Read/Grep/Glob/Bash/…). While **unresolved** (no matching `tool_result`): activity line from `getActivityDescription` + `…` + dim `(ctrl+o to expand)` in prompt; then dim **`  ⎿  `** hint (quoted pattern for Grep, path for Read, `commandAsHint`-style for Bash). Resolved tools drop the last two rows. |
| Builtin status (model · Context · $ · Debug) | [`cmd/gou-demo/builtin_status_line.go`](../../cmd/gou-demo/builtin_status_line.go) | Mirrors TS `BuiltinStatusLine` shape: model from last submit / env; **Context %** uses **[`conversation.Store`](../../gou/conversation/store.go) `Usage*Total`** when ccbstream **`usage`** lines were applied (same fields as protocol `input_tokens` / `output_tokens`), else falls back to message-size heuristic; window via **`GOU_DEMO_CONTEXT_WINDOW`** (default 200k); **`GOU_DEMO_SESSION_COST_USD`**; **Debug** when `GOU_DEMO_DEBUG` / `CLAUDE_CODE_DEBUG`. **`GOU_DEMO_NO_BUILTIN_STATUS=1`** disables the row. |
| Verbose tool JSON | `GOU_DEMO_VERBOSE_TOOL_OUTPUT` | Same env as tool_result preview cap; forces `formatNamedTool` JSON for `tool_use` / `server_tool_use`. |
| Dim `(ctrl+o to expand)` on tool rows | [`cmd/gou-demo/main.go`](../../cmd/gou-demo/main.go) | Only when `uiScreen == prompt` and not transcript dump mode; literal `ctrl+o` until shortcut display is wired. |
| `✻` + spinner verb + ellipsis animation | `queryBusy` + `spinner_verbs.go`, tick in `main.go` | Verbs list manually synced from TS `SPINNER_VERBS`. |
| Tip line priority | `effectiveSpinnerTip` in `spinner_tip.go` | Simplified: **30m** → `/clear` tip, **30s** → `/btw` tip (no `btwUseCount` gate), else fixed **prompt-queue** sentence from TS registry. No full `tipScheduler` / context tips. |
| Tips disabled | `CLAUDE_CODE_SPINNER_TIPS_ENABLED=0` or `GOU_DEMO_SPINNER_TIPS=0` | Default tips on. |

## Automated

- `go test ./gou/messagerow/...` — `tool_activity_test.go`, segment tests.
- `go test ./cmd/gou-demo/...` — `spinner_tip_test.go` and existing transcript/REPL tests.

## Related

- Transcript parity: [gou-demo-transcript-ts-parity.md](./gou-demo-transcript-ts-parity.md).
