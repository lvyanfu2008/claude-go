# CLAUDE.md

This file orients contributors and automation when working in **`claude-go/`** (Go module **`goc`**).

## No TypeScript runtime dependency

**`goc` builds and tests must not spawn Bun, Node, or `claude-code` `.ts` entrypoints** to implement product behavior (no `exec` of those in `go test`, `go run ./cmd/gou-demo`, or library defaults). Embedded JSON, markdown, and generated literals copied from the TS product are **static assets**, not a runtime dependency. Comments may cite TS paths for parity. Optional **manual** comparisons (e.g. a sibling checkout running `dump-init-state`) are out of band and not required for CI.

## Layout

- **`toolparity/`** — Curated TS built-in list vs Go parity: edit **`catalog.json`**, then run **`go run ./cmd/gen-tool-parity`** (or **`go generate ./toolparity`**) to refresh **`TS_GO_TOOL_PARITY.md`**.
- **Todo v2 (`TaskCreate` / `TaskGet` / `TaskList` / `TaskUpdate`)** — Implemented under **`ccb-engine/paritytools/tasks_v2.go`** (config-home `tasks/<listId>/`, same layout as TS `tasks.ts`). Omitted vs TS: task-created/completed hooks, teammate mailbox, verification nudge, and in-process teammate task-list resolution (see `TODO(ts parity)` in that file). Enable in non-interactive runs with **`CLAUDE_CODE_ENABLE_TASKS=1`**. **`commands.TodoV2Enabled`** gates the model-facing list: embedded **`commands/data/tools_api.json`** includes both `TodoWrite` and the four `Task*` tools; **`toolpool.FilterToolsByPerToolEnabled`** hides `TodoWrite` and shows `Task*` when v2 is on (mirrors TS `getTools` + `isEnabled`). **`toolpool.GetTools`** also mirrors TS **`getAllBaseTools` / `isEnabled`** for Cron (`CLAUDE_CODE_DISABLE_CRON`), plan mode under Kairos+channels, `TaskOutput` when `USER_TYPE=ant`, **`tstenv.ToolSearchEnabledOptimistic`**, agent-swarm tools, and strips **Glob/Grep** when **`EMBEDDED_SEARCH_TOOLS`** matches **`hasEmbeddedSearchTools`**. Streaming parity / gou-demo with **`GOU_DEMO_USE_EMBEDDED_TOOLS_API`** (see **`gou/pui/params.go`**) uses that path.
- **`anthropicmessages/`** — HTTP SSE client for Anthropic Messages (`PostStream`, stream parsing). **`BetasForToolsJSON`** mirrors the ToolSearch `anthropic-beta` gate from TS / `ccb-engine/internal/toolsearch` without importing engine internals.
- **`conversation-runtime/query/`** — Port of `src/conversation-runtime/query.ts`: compaction, `Query`, optional host **`QueryDeps.CallModel`**, and optional **streaming parity** (`runStreamingParityModelLoop`: Anthropic SSE + `streamingtool` + `toolexecution`). Streaming uses **`goc/anthropicmessages.PostStream`**, which honors **`CLAUDE_CODE_LOG_API_REQUEST_BODY`** / **`CLAUDE_CODE_LOG_API_RESPONSE_BODY`** via **`ccb-engine/apilog`** (request JSON + raw SSE body, capped for response size).
- **`conversation-runtime/process-user-input/`** — `processUserInput` port; **`ApplyQueryHostEnvGates`** / **`WireToolexecutionFromProcessUserInput`** hook hosts that call `query.Query` after `ShouldQuery`.
- **`toolexecution/`** — Tool execution aligned toward `toolExecution.ts`: **`PermissionDecision`** + **`QueryCanUseTool`**, **`ResolveHookPermissionDecision`**, **`CheckRuleBasedPermissions`** (alwaysDeny / alwaysAsk via **`goc/permissionrules`** when **`ToolPermission`** / **`ExecutionDeps.ToolPermission`** is set), **`RunToolUseChan`** (rules after query gate), **`CheckPermissionsAndCallTool`** (pre-hook + optional **`PreToolHookPermission`** + early **JSON schema** validation for registry tools), **`AskResolver`** for headless `ask`.
- **`query.QueryParams.ToolPermissionContext`** — copied into **`ToolexecutionDeps.ToolPermission`** on streaming parity ([`streaming_loop.go`](claude-go/conversation-runtime/query/streaming_loop.go)); gou-demo forwards **`ProcessUserInputContextData.ToolPermissionContext`** when set.
## Permissions / `canUseTool` (vs TS)

- **`query.CanUseToolFn`** is **`toolexecution.QueryCanUseToolFn`**: returns **`PermissionDecision`** (`allow` / `deny` / `ask`) + `error`. Legacy `(bool, error)` hosts use **`toolexecution.LegacyBoolQueryGate`** (see **`process-user-input/query_wire.go`**).
- **`ask`**: the library does not render UI. Set **`ExecutionDeps.AskResolver`** to map `ask` → allow/deny, or rely on the default headless deny. **gou-demo** sets **`AskResolver` → allow** when **`GOU_QUERY_ASK_STRATEGY=allow`** (streaming parity path only).
- **`NewStreamingToolExecutor`** receives the same **`QueryCanUseToolFn`** as **`QueryParams.CanUseTool`** so the executor’s `canUseTool` slot is no longer always nil (**`run_tool_use_runner`** overlays the executor argument onto deps for each tool run).

## Model paths (important)

| Path | When | Notes |
|------|------|--------|
| **Streaming parity** (`query.Query` with `StreamingParity` + host gate) | [StreamingParityPathEnabled](conversation-runtime/query/query_config_build.go) is always true; set `QueryParams.StreamingParity`, plus provider keys (e.g. Anthropic) for real HTTP. `GOU_QUERY_STREAMING_PARITY=1` still sets [QueryConfigGates.StreamingParityPath](conversation-runtime/query/config.go) for diagnostics. | Inside **`queryLoop`**, if `useStream` is true it **runs before** `CallModel`. **gou-demo** uses this for real turns when the gate and key are set; otherwise it falls back to a simulated stream (or `-fake-stream`). |
| **CallModel** (`QueryDeps.CallModel` set) | Host supplies a non-streaming or custom model loop | Runs when streaming parity is off or not selected. |

Env gates are assembled in **`query.BuildQueryConfig`** (`query_config_build.go`). Project **`.claude/settings.go.json`** `env` block is merged at runtime (see **`ccb-engine/settingsfile`**, gou-demo init).

## Commands

```bash
cd claude-go
go test ./...
go build -o /dev/null ./cmd/gou-demo
```

## Related docs

- Plan: streaming parity + toolexecution roadmap (repo `.cursor/plans/`, do not edit plan files from automation unless asked).
- TS mirror rule: `.cursor/rules/claude-go-mirror-typescript.mdc` when porting behavior.
