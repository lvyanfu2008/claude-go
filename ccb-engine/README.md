# ccb-engine

Headless conversation engine for **Go-hosted** orchestration (迁移期). Lives under `goc/ccb-engine/` (Go module root is **`goc/go.mod`**; `ccb-engine` is a package path `goc/ccb-engine/...`, not a nested module). **Ink REPL 计划废弃**（见 [architecture-go-orchestration.md](../../docs/plans/architecture-go-orchestration.md)）；**Bun/Ink REPL 不再** spawn `ccb-engine` 或编排整轮。TS [`src/goEngine/client.ts`](../../src/goEngine/client.ts) 的 `submitTurn` 仍用于单测与未来 **Go 所连 TS 工具 worker**（`onExecuteTool` → `executeCcbBridgeToolUse`）。

**权威产品架构**：[architecture-go-orchestration.md](../../docs/plans/architecture-go-orchestration.md)。历史链接：[architecture-go-ts-strategy.md](../../docs/plans/architecture-go-ts-strategy.md)；本引擎细节亦见 [architecture-go-middle-layer.md](../../docs/plans/architecture-go-middle-layer.md)。

**路线里程碑索引**：[go-core-milestones.md](../../docs/plans/go-core-milestones.md)（若存在；否则以 `architecture-go-orchestration.md` 为准）。

**V6 批 927–930 / 1431–1440**：与根 README、[architecture-go-orchestration.md](../../docs/plans/architecture-go-orchestration.md)、[architecture-go-ts-strategy.md](../../docs/plans/architecture-go-ts-strategy.md)、[architecture-go-middle-layer.md](../../docs/plans/architecture-go-middle-layer.md)、[`CLAUDE.md`](../../CLAUDE.md) Core Loop 交叉引用 **复核仍真**（见 [v6-module-boundaries.md](../../docs/plans/v6-module-boundaries.md) 当前窗）。

**Socket 协议与宿主**

1. **无 TUI 自动化**（listener）：`go build -o ccb-socket-host ./cmd/ccb-socket-host`，然后 `ccb-socket-host -socket $CCB_ENGINE_SOCKET`（或只设 `CCB_ENGINE_SOCKET`）。见仓库根 [`scripts/ccb-worker-daemon.sh`](../../scripts/ccb-worker-daemon.sh)（`start` = socket-host + worker；**`start-worker`** = 仅 TS worker，须 socket 已由 **ccb-socket-host** 等监听；**`stop-worker`** 只停 worker）。
2. **gou-demo** 不再内嵌 socketserve、不 spawn **`ccb-engine-tool-worker`**；TUI 真实模型走 **`conversation-runtime/query` 流式 parity**（Anthropic SSE 等），或 `-fake-stream` 纯模拟。
3. 遗留 **`CLAUDE_CODE_CCB_ENGINE`** 门控已从 **Ink/Bun REPL** 移除。若自定义 TS 进程调用 `submitTurn`，可自行读环境变量；连接超时见下。

Optional timeouts (TS `submitTurn` client): `CLAUDE_CODE_CCB_ENGINE_CONNECT_TIMEOUT_MS` (default 15000).（历史名 `CLAUDE_CODE_CCB_ENGINE_TURN_TIMEOUT_MS` 曾用于已删除的 REPL 整轮超时，现无 TS 消费方。）

### TS worker：`ccb-engine-tool-worker`

在仓库根（含本仓库 `package.json`）：

1. 先保证 **Go 侧**在 `CCB_ENGINE_SOCKET` 上 **listen**（**`ccb-socket-host`** / `scripts/ccb-worker-daemon.sh start`）。
2. **`printf '%s\n' '<one-line SubmitUserTurn JSON>' | bun run ccb-engine-tool-worker $CCB_ENGINE_SOCKET`**（或 `npm run ccb-engine-tool-worker`）。stdin 为与 [spec/protocol-v1.md](spec/protocol-v1.md) 一致的 **`{"method":"SubmitUserTurn","id":"…","payload":{…}}`**；stdout 为 NDJSON 流（与 Go 侧下发事件同形，含 `execute_tool` 时 worker 写回 `ToolResult`）。
3. 环境：**`CCB_WORKER_CWD`**（可选）在启动前 `chdir`。**`CCB_WORKER_STDIN_LOOP=1`**：stdin **多行**（每行一个 SubmitUserTurn），**EOF** 结束；只 **`init` / bootstrap 一次**，复用 store/tools。**`CCB_GO_BRIDGE_THIN_TOOL_EXECUTION`** / **`CCB_GO_BRIDGE_TRUST_GO_GATE`** 默认在 worker 内置为 `1`（可覆盖）。

**TS 侧工具执行（Go 桥 env）**：`CCB_GO_BRIDGE_THIN_TOOL_EXECUTION`、`CCB_GO_BRIDGE_TRUST_GO_GATE` 等由 **worker 进程**读取（见 [`src/goEngine/ccbGoBridgeEnv.ts`](../../src/goEngine/ccbGoBridgeEnv.ts)）。手动启动 worker 时在 shell 或项目 **`settings.json`** 中配置；兼容旧名 `CLAUDE_CODE_CCB_THIN_*`。

## Build

```bash
cd goc
go build -o ccb-engine ./ccb-engine/cmd/ccb-engine
go build -o ccb-socket-host ./cmd/ccb-socket-host
```

## Test

```bash
cd goc
go test ./ccb-engine/...
```

Mocked API tests run with `go test ./ccb-engine/...` from `goc/`. Live API:

```bash
cd goc
go test -tags=integration ./ccb-engine/internal/engine/ -count=1
```

(requires `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` unless supplied via project settings; see below)

### Project `.claude/settings.go.json` (`env`)

On startup, **`ccb-engine`** and **`gou-demo`** call [`settingsfile.EnsureProjectClaudeEnvOnce()`](settingsfile/ensure.go), which merges `env` from (later wins on duplicate keys; **Go never reads project `.claude/settings.json`** — that file is **TypeScript CLI only**):

1. **User:** `$CLAUDE_CONFIG_DIR/settings.json` if **`CLAUDE_CONFIG_DIR`** is set, else **`~/.claude/settings.json`** (matches TS `getClaudeConfigHomeDir` + `settings.json`).
2. **Project root for Go:** **`CCB_ENGINE_PROJECT_ROOT`** if set; otherwise the **nearest ancestor of the current working directory** whose `.claude/` contains **`settings.go.json`** or **`settings.local.json`**. If neither marker exists, **`projectRoot`** is the **abs path of the starting cwd** (no upward walk to a TS-only `.claude/settings.json`).
3. **Project files merged into env:** `<projectRoot>/.claude/settings.go.json`, then `<projectRoot>/.claude/settings.local.json` (machine-specific / gitignored overrides).

Any variable already non-empty in the process environment is left unchanged (shell / parent wins).

- It reads the top-level **`env`** object and calls `os.Setenv` for each key.
- **Existing non-empty** environment variables are **not** overwritten (shell / parent process wins).
- This is still a **subset** of Claude Code’s full settings merge (no managed policy / remote sync / trust UI). Env keys are the main use case.

`ccb-engine` spawned by a **遗留 TS 宿主** inherits the parent `process.env` first; the TS host may have merged project **`settings.json`** into that env before spawning — Go **`EnsureProjectClaudeEnvOnce`** still applies **`settings.go.json` / `settings.local.json`** for keys that remain empty.

### Tool input JSON Schema (before `execute_tool` → TS)

On each `tool_use`, **`Session.RunTurn`** validates the model’s `input` against the matching entry in the turn’s `tools` list (`input_schema`) **before** invoking `ToolRunner` (so in bridge mode, before emitting **`execute_tool`**). Validation uses [`github.com/santhosh-tekuri/jsonschema/v6`](https://github.com/santhosh-tekuri/jsonschema) with **draft-07** as the default when `$schema` is absent. On failure the engine appends a **`tool_result`** with `is_error` and does not call the runner.

### Optional Go allowlist (`permission_context`)

When **`CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1`**, the engine requires **`SubmitUserTurn` payload `permission_context`** JSON with a non-empty **`allowedTools`** string array. If the model’s `tool_use.name` is not in that list, the engine appends an error **`tool_result`** and does **not** emit **`execute_tool`**. When a tool passes both schema and allowlist, **`execute_tool`** includes optional **`policy`**: `{ "decision":"allow", "source":"ccb-engine" }` so the TS client can use a trust-only execution path (see `docs/plans/go-policy-ts-pure-execution.md`, `goc/ccb-engine/internal/toolpolicy`).

`ccb-engine` speaks the **Anthropic Messages API** (`/v1/messages`, `x-api-key`, `anthropic-version`). An **OpenAI-compatible** base URL (e.g. some third-party gateways) may return errors until a separate adapter exists; use an Anthropic-compatible endpoint or the official Anthropic host for live tests.

`NewClient()` resolves model id via [`goc/modelenv.ResolveWithFallback`]: `CCB_ENGINE_MODEL` → `ANTHROPIC_MODEL` → `ANTHROPIC_DEFAULT_SONNET_MODEL` → `ANTHROPIC_DEFAULT_HAIKU_MODEL` → `ANTHROPIC_DEFAULT_OPUS_MODEL` → default `claude-sonnet-4-20250514`. OpenAI-compat (`newOpenAICompatFromEnv`) uses the same chain with fallback `deepseek-chat`.

### DeepSeek / OpenAI-compatible APIs

`ccb-engine` selects **`POST …/chat/completions`** (Bearer token) when any of these is true:

- `CCB_ENGINE_LLM=openai` (or `deepseek` / `openai-compat`)
- `CLAUDE_CODE_USE_OPENAI=1`
- `ANTHROPIC_BASE_URL` contains `deepseek` (case-insensitive)

You can put the same variables in the project **`.claude/settings.go.json`** `env` block (merged before the LLM runs for pure Go hosts) or **export them in the shell**. Typical DeepSeek setup:

- `ANTHROPIC_BASE_URL=https://api.deepseek.com/v1` (must include `/v1`; requests go to `…/v1/chat/completions`)
- `ANTHROPIC_AUTH_TOKEN` or `OPENAI_API_KEY` = your key
- `CCB_ENGINE_MODEL=deepseek-chat` (or `ANTHROPIC_MODEL` / `ANTHROPIC_DEFAULT_*_MODEL` via `goc/modelenv`)

`cmd/ccb-engine` uses `llm.NewFromEnv()` so it follows the rules above automatically.

### LLM request/response body log (TS parity)

Same env switches as [`src/utils/debug.ts`](../../src/utils/debug.ts) `logLlmApiRequestBody` / `logLlmApiResponseBody`:

- `CLAUDE_CODE_LOG_API_REQUEST_BODY=1` — log JSON **request** body
- `CLAUDE_CODE_LOG_API_RESPONSE_BODY=1` — log JSON **response** body

Format: `timestamp [API_REQUEST_BODY|API_RESPONSE_BODY] <label>\n<compact JSON>\n----------\n`

**Output file** (first match): `CLAUDE_CODE_DEBUG_LOG_FILE`; else `CLAUDE_CODE_DEBUG_LOGS_DIR/ccb-engine-llm-api.txt`; else `$HOME/.claude/debug/ccb-engine-llm-api.txt`. **`ccb-engine` and `gou-demo` call `apilog.PrepareIfEnabled()` after loading project settings**, which creates `$HOME/.claude/debug/` (and an empty log file) as soon as either `CLAUDE_CODE_LOG_API_*` flag is on—so the directory exists even before the first HTTP call. The first prepare or append prints `[ccb-engine apilog] writing LLM API bodies to …` on stderr. These variables can be set in project `.claude/settings.go.json` `env` (merged on startup for Go; see above). To append next to the Bun session log, set `CLAUDE_CODE_DEBUG_LOG_FILE` to the same path TS uses.

**`gou-demo` caveat:** use **`-fake-stream`** (or **`GOU_DEMO_USE_FAKE_STREAM=1`**, or **`GOU_DEMO_CCB_INLINE=0`**) only when you want a simulated stream with **no** LLM. For real HTTP + apilog bodies, set **Anthropic** (or configured provider) keys and **`GOU_QUERY_STREAMING_PARITY=1`** or **`GOU_DEMO_STREAMING_TOOL_EXECUTION=1`**.

**Empty `~/.claude/debug/`:** if **`CLAUDE_CODE_LOG_API_REQUEST_BODY`** / **`CLAUDE_CODE_LOG_API_RESPONSE_BODY`** are not set, **`PrepareIfEnabled` does not create** `ccb-engine-llm-api.txt` (the directory may exist from another tool or be empty). Set one or both flags in the shell or in **project** `.claude/settings.go.json` `env`. Run with **`CLAUDE_CODE_APILOG_DIAG=1`** to print the resolved path and flag state to stderr.

## Hooks (command)

Package [`internal/hooks`](internal/hooks/exec.go) runs `sh -c` hooks with JSON on stdin and optional **PromptRequest** lines on stdout, matching Claude Code’s [`hooks.ts`](../../src/utils/hooks.ts) protocol. See [`spec/protocol-v1.md`](spec/protocol-v1.md).

## Run (smoke)

```bash
export ANTHROPIC_API_KEY=...   # or ANTHROPIC_AUTH_TOKEN
# optional: export ANTHROPIC_BASE_URL=...  export CCB_ENGINE_MODEL=...
./ccb-engine -prompt "Say hello in one sentence."
```

## Unix socket protocol (`socketserve`)

The **listener** is not a subcommand of **`ccb-engine`** (smoke CLI only: `-prompt`). Production paths:

- **Headless:** `ccb-socket-host` (`go build -o ccb-socket-host ./cmd/ccb-socket-host`), e.g. `ccb-socket-host -socket /tmp/ccb.sock` or `CCB_ENGINE_SOCKET=/tmp/ccb.sock ccb-socket-host`.

Wire-up summary:

- **Client → Go:** one JSON line to start a turn: `{"method":"SubmitUserTurn","id":"<uuid>","payload":{"text":"...","messages":[...],"tools":[...]}}`. Optional `messages` replaces the engine transcript (API-shaped user/assistant messages); optional `tools` is forwarded to the model as the tool list. During an active turn the client may send additional lines: **`ToolResult`** (reply to `execute_tool`) and **`CancelTurn`** (abort).
- **Go → client:** NDJSON `StreamEvent` lines (same shapes as `-json-events` on stderr for the smoke CLI), including **`execute_tool`** when the model requests a tool and the bridge is active, then a final `{"type":"response_end","id":"<same as request>"}` line per request.
- When `tools` is omitted, the engine falls back to **stub tools** (`DefaultStubTools`). When `tools` is provided, the TS client is expected to execute tools via **`execute_tool`** round-trips on the same socket.

Implementation note: the `BridgeRunner` write callback must **not** take the same mutex that `BridgeRunner.Run` already holds while emitting `execute_tool` (double-lock deadlock). [`socketserve.HandleConn`](socketserve/socketserve.go) uses a separate `writeEvBridge` that writes NDJSON without re-locking.

**Session model:** each `SubmitUserTurn` creates a **fresh** Go session and replaces its transcript from `messages` when provided; cross-turn history lives in TS, not in Go. **`GetSnapshot` / `client_state_rev` reconciliation** in the spec targets a possible future **persistent** engine session, not this hydrate-per-turn path (see [`spec/protocol-v1.md`](spec/protocol-v1.md)).

When a **遗留 TS 宿主** connects to the socket, it may set **`CCB_ENGINE_PROJECT_ROOT`** so project **`.claude/settings.go.json`** (and Go env merge) is resolved consistently.

Environment variables for the LLM are the same as the smoke run (`ANTHROPIC_*`, `CCB_ENGINE_*`, OpenAI-compat flags, etc.). For **Go**, project `env` from **`.claude/settings.go.json`** / **`settings.local.json`** applies when `CCB_ENGINE_PROJECT_ROOT` points at the project (see above).

## Protocol

See [`spec/protocol-v1.md`](spec/protocol-v1.md) and [`spec/protocol-v1.schema.json`](spec/protocol-v1.schema.json) for the TS↔Go v1 JSON-RPC-style shapes and stream events.
