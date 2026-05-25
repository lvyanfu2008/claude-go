# HARNESS.md

This file provides guidance to Harness Code (harness.ai/code) when working with code in this repository.

## Hard constraint

**No TypeScript runtime dependency**: builds/tests must not spawn Bun, Node, or `harness-code` `.ts` entrypoints. Embedded JSON/markdown from the TS codebase are static assets only.

## Commands

```bash
go test ./...
go vet ./...
go build -o /dev/null ./cmd/harness
golangci-lint run ./...
```

## go:generate

After syncing TS JSON exports (out-of-band, not CI), regenerate:

```bash
go generate ./tools/toolparity
go generate ./commands/handwritten
```

## Key details

- Module name is `goc` (not `harness-go`).
- `canUseTool` / `QueryCanUseToolFn` returns `PermissionDecision` (`allow` / `deny` / `ask`) — 3-state, not a bool.
- Branch naming uses conventional prefixes (e.g., `feature/`, `fix/`, `base-`).

## Project structure

| Directory | Purpose |
|-----------|---------|
| `conversation-runtime/query/` | LLM query loops: Anthropic (`streaming_loop.go`), OpenAI (`openai_stream_loop.go`), Gemini, non-streaming |
| `tools/` | Tool implementations (Bash, Read, Write, Edit, TaskCreate, etc.) |
| `tools/toolpool/` | Tool registry, wire specs, enable/disable gating |
| `internal/zoglayer/` | Zog validators for all tools |
| `commands/` | Slash command handlers |
| `gou/` | Bubble Tea TUI: app, conversation store, message rendering, markdown |
| `gou/app/` | TUI model, update handlers, view rendering, streaming display |
| `gou/message/` | Message renderers (assistant, tool_use, tool_result, thinking) |
| `gou/conversation/` | Store (messages, StreamingText, StreamingToolUses) |
| `ccb-engine/` | Legacy CCB stream engine (not used in streaming parity path) |
| `appstate/` | Runtime state, env var constants |
| `agents/` | Sub-agent definitions (guide, explore, plan, etc.) |

## Streaming architecture

Two parallel streaming paths:

1. **Anthropic path** (`streaming_loop.go`): HTTP SSE via Messages API `PostStream` → `ProcessStreamPayloads`. Yields `StreamEvent` (content_block_delta) for incremental display.
2. **OpenAI path** (`openai_stream_loop.go`): HTTP SSE via `/v1/chat/completions` with `stream:true`. Same yield pattern via `assistantStreamAccumulator`.

Both paths yield `QueryYield{StreamEvent: ev.Raw}` for `content_block_delta` events, which flows through:
- `runQueryStreamingParityTurn` → `programSend(gouStreamEventMsg{Raw: ...})`
- `handleUpdateGouStreamEvent` → `store.AppendStreamingChunk(delta)` for `text_delta` / `thinking_delta`

Streaming text is rendered in the message pane (above completed messages), not in the bottom prompt area. `promptBottomStreamRows()` reserves space but no longer duplicates streaming content.

## Key env vars

| Var | Effect |
|-----|--------|
| `HARNESS_CODE_ENABLE_TASKS=1` | Enable Todo v2 (TaskCreate/Get/List/Update); default on in interactive mode |
| `HARNESS_CODE_USE_OPENAI=1` | Route LLM queries through OpenAI-compatible API |
| `HARNESS_CODE_USE_GROK=1` | Route LLM queries through Grok API |
| `OPENAI_API_KEY` | API key for OpenAI-compatible providers |
| `OPENAI_BASE_URL` | Base URL override for OpenAI-compatible API |
| `OPENAI_ENABLE_THINKING=1` | Enable thinking mode for OpenAI-compatible providers |
| `GO_TOOL_INPUT_VALIDATOR=zog` | Use zog-validated Bash (BashZog) |
| `CCB_ENGINE_MODEL` | Model ID (e.g., `deepseek-v4-pro`) |
| `GOU_QUERY_OPENAI_CHAT_NO_STREAM=1` | Force non-streaming mode for OpenAI path |
| `HARNESS_CODE_NON_INTERACTIVE=1` | Disable interactive features (disables Todo v2) |

## Key patterns

- **No stubs**: Always implement with DI (e.g., `WriteDeps` pattern), never use `ParityStub`.
- **Edit tool**: Use `Edit` tool for code changes. If it fails on tab-indented Go code, use Python scripts (never `sed -i` on Go files).
- **Debug logging**: Use `logForDebugging` / existing debug infrastructure, not ad-hoc file writes or stderr.
- **Go testing parity**: Go-side code changes must include corresponding test code, matching TS-side expectations.
- **Verify before acting**: Always verify implementation claims against actual code before acting on them.
