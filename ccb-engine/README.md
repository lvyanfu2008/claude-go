# ccb-engine

Shared **libraries** under `goc/ccb-engine/...` used by the **gou-demo default path**: [`goc/conversation-runtime/query`](../conversation-runtime/query/) (HTTP streaming parity), [`goc/gou/ccbstream`](../gou/ccbstream/) (NDJSON events → `conversation.Store`), [`goc/toolpool` / `toolexecution`](../toolexecution/) with [`skilltools.ParityToolRunner`](skilltools/parity_runner.go), plus [`apilog`](apilog/) / [`diaglog`](diaglog/) for debug parity.

There is **no** in-repo `ccb-engine` CLI or `Session.RunTurn` loop anymore; interactive chat history is **not** driven from here.

**Product architecture:** [architecture-go-orchestration.md](../../docs/plans/architecture-go-orchestration.md).

## Test

```bash
cd goc
go test ./ccb-engine/...
```

### Project `.claude/settings.go.json` (`env`)

On startup, packages here and **`gou-demo`** call [`settingsfile.EnsureProjectClaudeEnvOnce()`](settingsfile/ensure.go), which merges `env` from (later wins on duplicate keys; **Go never reads project `.claude/settings.json`** — that file is **TypeScript CLI only**):

1. **User:** `$CLAUDE_CONFIG_DIR/settings.json` if **`CLAUDE_CONFIG_DIR`** is set, else **`~/.claude/settings.json`**.
2. **Project root for Go:** **`CCB_ENGINE_PROJECT_ROOT`** if set; otherwise the **nearest ancestor of cwd** whose `.claude/` contains **`settings.go.json`** or **`settings.local.json`**.
3. **Project files merged into env:** `<projectRoot>/.claude/settings.go.json`, then `<projectRoot>/.claude/settings.local.json`.

Existing non-empty environment variables are **not** overwritten.

### DeepSeek / OpenAI-compat env (used by query / `modelenv`)

Same flags as elsewhere in `goc/`: e.g. `CCB_ENGINE_LLM=openai`, `CLAUDE_CODE_USE_OPENAI=1`, or `ANTHROPIC_BASE_URL` containing `deepseek`. Model id chain: [`goc/modelenv.ResolveWithFallback`](../modelenv/model_env.go).

### LLM request/response body log (TS parity)

Same env as TS `logLlmApiRequestBody` / `logLlmApiResponseBody`:

- `CLAUDE_CODE_LOG_API_REQUEST_BODY=1`
- `CLAUDE_CODE_LOG_API_RESPONSE_BODY=1`

Output path resolution: [`debugpath.ResolveLogPath`](debugpath/path.go). **`apilog.PrepareIfEnabled()`** runs after merged settings (e.g. from **`gou-demo`**). Diagnostics block: **`GOU_DEMO_LOG=1`** (same as gou-demo trace) → [`apilog.MaybePrintDiag`](apilog/apilog.go); log lines are tagged **`[GOU_DEMO_LOG_APILOG_DIAG]`**.

**`gou-demo`:** use **`-fake-stream`** / **`GOU_DEMO_USE_FAKE_STREAM=1`** only for simulation without HTTP. For real requests + body logs, set API keys; streaming parity is enabled by the query host when **`QueryParams.StreamingParity`** is set (optional **`GOU_QUERY_STREAMING_PARITY=1`** sets config flags for tooling).

## Packages (non-exhaustive)

| Area | Packages |
|------|-----------|
| HTTP / tools wire (query path) | [`internal/anthropic`](internal/anthropic/), [`internal/toolsearch`](internal/toolsearch/), [`toolsearchwire`](toolsearchwire/) |
| Local tool parity | [`skilltools`](skilltools/), [`localtools`](localtools/), [`paritytools`](paritytools/), [`bashzog`](bashzog/), [`toolstub`](toolstub/) |
| Settings / paths | [`settingsfile`](settingsfile/), [`debugpath`](debugpath/) |
| Logging | [`apilog`](apilog/), [`diaglog`](diaglog/) |

NDJSON **`StreamEvent`** handling for the TUI lives in [`gou/ccbstream`](../gou/ccbstream/event.go).
