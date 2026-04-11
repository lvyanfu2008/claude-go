# CLAUDE.md

This file orients contributors and automation when working in **`claude-go/`** (Go module **`goc`**).

## Layout

- **`toolparity/`** — Curated TS built-in list vs Go parity: edit **`catalog.json`**, then run **`go run ./cmd/gen-tool-parity`** (or **`go generate ./toolparity`**) to refresh **`TS_GO_TOOL_PARITY.md`**.
- **Todo v2 (`TaskCreate` / `TaskGet` / `TaskList` / `TaskUpdate`)** — Implemented under **`ccb-engine/paritytools/tasks_v2.go`** (config-home `tasks/<listId>/`, same layout as TS `tasks.ts`). Omitted vs TS: task-created/completed hooks, teammate mailbox, verification nudge, and in-process teammate task-list resolution (see `TODO(ts parity)` in that file). Enable in non-interactive runs with **`CLAUDE_CODE_ENABLE_TASKS=1`**.
- **`anthropicmessages/`** — HTTP SSE client for Anthropic Messages (`PostStream`, stream parsing). **`BetasForToolsJSON`** mirrors the ToolSearch `anthropic-beta` gate from TS / `ccb-engine/internal/toolsearch` without importing engine internals.
- **`conversation-runtime/query/`** — Port of `src/conversation-runtime/query.ts`: compaction, `Query`, optional **`LocalTurnCallModel`** (ccb-engine), and optional **streaming parity** (`runStreamingParityModelLoop`: Anthropic SSE + `streamingtool` + `toolexecution`).
- **`conversation-runtime/process-user-input/`** — `processUserInput` port; **`ApplyQueryHostEnvGates`** / **`WireToolexecutionFromProcessUserInput`** hook hosts that call `query.Query` after `ShouldQuery`.
- **`toolexecution/`** — Tool execution aligned toward `toolExecution.ts`: **`PermissionDecision`** + **`QueryCanUseTool`**, **`ResolveHookPermissionDecision`**, **`CheckRuleBasedPermissions`** (alwaysDeny / alwaysAsk via **`goc/permissionrules`** when **`ToolPermission`** / **`ExecutionDeps.ToolPermission`** is set), **`RunToolUseChan`** (rules after query gate), **`CheckPermissionsAndCallTool`** (pre-hook + optional **`PreToolHookPermission`** + early **JSON schema** validation for registry tools), **`AskResolver`** for headless `ask`.
- **`query.QueryParams.ToolPermissionContext`** — copied into **`ToolexecutionDeps.ToolPermission`** on streaming parity ([`streaming_loop.go`](claude-go/conversation-runtime/query/streaming_loop.go)); gou-demo forwards **`ProcessUserInputContextData.ToolPermissionContext`** when set.
- **`ccb-engine/internal/engine`** — Optional **`ToolexecutionRunner`** implements **`ToolRunner`** via **`RunToolUseChan`** so localturn can share the same permission/execution path as query streaming (A5).

## Permissions / `canUseTool` (vs TS)

- **`query.CanUseToolFn`** is **`toolexecution.QueryCanUseToolFn`**: returns **`PermissionDecision`** (`allow` / `deny` / `ask`) + `error`. Legacy `(bool, error)` hosts use **`toolexecution.LegacyBoolQueryGate`** (see **`process-user-input/query_wire.go`**).
- **`ask`**: the library does not render UI. Set **`ExecutionDeps.AskResolver`** to map `ask` → allow/deny, or rely on the default headless deny. **gou-demo** sets **`AskResolver` → allow** when **`GOU_QUERY_ASK_STRATEGY=allow`** (streaming parity path only).
- **`NewStreamingToolExecutor`** receives the same **`QueryCanUseToolFn`** as **`QueryParams.CanUseTool`** so the executor’s `canUseTool` slot is no longer always nil (**`run_tool_use_runner`** overlays the executor argument onto deps for each tool run).

## Two model paths (important)

| Path | When | Notes |
|------|------|--------|
| **LocalTurn** (`localturn.RunSubmitUserTurn` / `QueryDeps.CallModel` = `LocalTurnCallModel`) | gou-demo default when streaming parity env is off or no Anthropic key | Uses ccb-engine stream events (`ccbstream`). |
| **Streaming parity** (`query.Query` with `StreamingParity` + env gate) | `GOU_QUERY_STREAMING_PARITY=1` **or** `GOU_DEMO_STREAMING_TOOL_EXECUTION=1`, plus `QueryParams.StreamingParity`, plus Anthropic key for real HTTP | Inside **`queryLoop`**, if `useStream` is true it **runs before** `CallModel` — so parity **wins** over LocalTurn when both are configured. gou-demo explicitly chooses this branch when the gate and key are set (preempts localturn for that submit). |

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
