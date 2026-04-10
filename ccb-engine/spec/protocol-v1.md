# ccb-engine protocol v1

Version string: `ccb-engine-v1` (send in `hello` / envelope for future socket transport).

## stateRev (cache coherence)

Monotonic **logical revision** of the canonical session in Go. Every mutation to canonical `messages` (user turn, assistant reply, tool results, compact) **must** bump `stateRev`.

### Session model (`serve` + Claude Code TS today)

Each **`SubmitUserTurn`** on a connection builds a **new** Go `Session` (see [`session.go`](../internal/engine/session.go)), optionally **`HydrateFromMessages`** from `payload.messages`, then runs one turn. **Prior turns are not kept inside Go**; the TypeScript client is the **source of truth** for the transcript and resends an API-shaped snapshot each submit. In this mode, `turn_complete.state_rev` is meaningful **within that turn only**; **`client_state_rev`** on the request is accepted for forward compatibility but **not used for mismatch recovery** yet.

### Durable Go session (optional future)

If the engine later holds a **single long-lived** canonical session across submits, TS should send the last seen `stateRev` as **`client_state_rev`**; on unexpected divergence the client would call **`GetSnapshot`** (see below). That flow is **not** required for the per-turn hydrate integration above.

## Error codes (`code` field)

| Code | Meaning |
|------|---------|
| `version_mismatch` | Client protocol version not supported |
| `invalid_request` | Malformed JSON or missing fields |
| `turn_in_progress` | New submit while a turn is running |
| `tool_unknown` | Tool name not registered (future TS bridge) |
| `api_error` | Upstream LLM API failure |
| `hook_timeout` | Hook subprocess exceeded timeout |
| `hook_failed` | Hook exited non-zero |

## Envelope (future bidirectional socket)

```json
{
  "v": "ccb-engine-v1",
  "id": "uuid",
  "method": "SubmitUserTurn",
  "payload": { }
}
```

Response:

```json
{
  "v": "ccb-engine-v1",
  "id": "uuid",
  "ok": true,
  "payload": { }
}
```

Or error:

```json
{
  "v": "ccb-engine-v1",
  "id": "uuid",
  "ok": false,
  "error": { "code": "api_error", "message": "..." }
}
```

## SubmitUserTurn

**Payload**

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Optional. User text appended after `messages` hydrate (omit if the new user turn is already the last entry in `messages`). |
| `messages` | array (optional) | API-shaped `[{ "role":"user"|"assistant", "content": ... }]` — same JSON model as Go `anthropic.Message`. When present, replaces the engine session transcript before `text` is appended. |
| `tools` | array (optional) | Anthropic `tools[]` entries (`name`, `description`, `input_schema`). When present, used for the model request instead of stub-only tools. |
| `permission_context` | object (optional) | Opaque JSON for Go-side tool policy. When **`CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1`**, the engine requires a non-empty **`allowedTools`** string array and rejects `tool_use` whose `name` is not listed (before `execute_tool`). Typical shape: `{ "allowedTools": ["Read","Bash",...], "permissionMode": "..." }`. See `docs/plans/go-policy-ts-pure-execution.md`. |
| `client_state_rev` | number (optional) | Last `stateRev` seen by client (reserved for snapshot mismatch handling). |

**Effect:** Hydrate session from `messages` (if any), append `text` (if non-empty), run model loop until `end_turn` or max tool rounds. At least one of `messages` or `text` must provide content.

## CancelTurn

One JSON line while a turn is in flight:

```json
{ "method": "CancelTurn", "id": "<request id optional>" }
```

Cancels the engine `context` for the active `SubmitUserTurn` (in-flight LLM and tool bridge waits).

## StreamEvent (server → client)

Union by `type`:

- `assistant_delta` — `{ "type":"assistant_delta", "text": "..." }`
- `tool_use` — `{ "type":"tool_use", "id":"...", "name":"...", "input": {} }`
- `tool_result` — `{ "type":"tool_result", "tool_use_id":"...", "content": "...", "is_error": false }` (optional `is_error`; emitted after a bridged tool completes)
- `usage` — `{ "type":"usage", "input_tokens": n, "output_tokens": n }`
- `turn_complete` — `{ "type":"turn_complete", "state_rev": n, "stop_reason": "..." }`
- `response_end` — `{ "type":"response_end", "id":"..." }` — terminates the NDJSON stream for one `SubmitUserTurn` request (`id` matches the request envelope).
- `error` — `{ "type":"error", "code":"...", "message":"..." }`
- `execute_tool` — `{ "type":"execute_tool", "call_id":"...", "tool_use_id":"...", "name":"...", "input":{}, "state_rev": n, "policy": { ... }? }` — client must reply with one **ToolResult** line on the same connection before the turn continues. Optional **`policy`** is set when **`CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1`** after the allowlist pass (e.g. `{ "decision":"allow", "source":"ccb-engine" }`) so the TS client can skip interactive `canUseTool` and use a trust-only execution path. **Note:** the engine may reject `input` earlier using JSON Schema (`tools[].input_schema`) or policy and emit a **`tool_result`** event without sending `execute_tool` (see ccb-engine README: `CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA`).

## ExecuteTool (Go → TS, `serve` mode)

Request from engine to client:

```json
{
  "type": "execute_tool",
  "call_id": "uuid",
  "tool_use_id": "toolu_...",
  "name": "Bash",
  "input": {},
  "state_rev": 42,
  "policy": { "decision": "allow", "source": "ccb-engine" }
}
```

(`policy` omitted when `CCB_ENGINE_ENFORCE_ALLOWED_TOOLS` is not set.)

## ToolResult (client → Go)

One JSON line on the same socket **during** an active `SubmitUserTurn` (after `execute_tool`, before `response_end`):

```json
{
  "call_id": "uuid",
  "tool_use_id": "toolu_...",
  "is_error": false,
  "content": "string or structured per API"
}
```

`call_id` must match the `execute_tool` event. Nested `SubmitUserTurn` requests are rejected until `response_end`.

## GetSnapshot

Response: full `messages` array (API-shaped) + `state_rev`.

**Note:** Intended for a **durable Go canonical session** (see [stateRev / session model](#staterev-cache-coherence)). With **per-turn hydrate** from TS, the client already has the transcript and typically does not need `GetSnapshot`.

## PromptRequest / PromptResponse (hooks, aligns with Claude Code)

**PromptRequest** (one JSON line on hook stdout):

```json
{
  "prompt": "request-id",
  "message": "Question for user",
  "options": [
    { "key": "yes", "label": "Yes", "description": "optional" }
  ]
}
```

**PromptResponse** (written to hook stdin, one line):

```json
{
  "prompt_response": "request-id",
  "selected": "yes"
}
```

## Sync hook output (optional)

Hooks may emit JSON on stdout; first line may be `{"async":true,...}` per Claude Code. See `internal/hooks` implementation notes in README.
