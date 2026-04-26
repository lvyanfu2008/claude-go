# `goc/commands/data/` (embed / tooling snapshots)

Built-in **`COMMANDS()`**, bundled skills, and builtin-plugin skills **do not live here** — listing metadata is authoritative in **`goc/commands/handwritten`** (see `handwritten_load.go`). Drift-check JSON is under **`../testdata/`** (`bun run export:builtin-commands`, `export:bundled-skills`, `export:builtin-plugin-skills`), then `cd goc && go run ./cmd/gencode-handwritten` after updating `builtin_commands_default.json`.

Additional prompt/skill rows can live under **`../builtin_*/`** (see [`../builtin_overlay/README.md`](../builtin_overlay/README.md)); `loadBuiltinCommands` merges them after the handwritten table (earlier names win on duplicate).

## Embedded

- **Channel / AskUserQuestion parity**: For TS parity when channel relay is active, set **`CLAUDE_CODE_GO_ALLOWED_CHANNELS`** (non-empty, comma-separated). In TS, channel tools also depend on `KAIROS` / `KAIROS_CHANNELS`; in **Go**, `featuregates.Feature("KAIROS")` is always true, so **`FEATURE_KAIROS_CHANNELS`** (or the env above) is what remains to mirror TS for AskUserQuestion omission.

## KAIROS (Go standalone vs `claude-code-best`)

| Topic | TypeScript | Go (`claude-go`) |
|-------|------------|------------------|
| Feature gate | Build-time `feature('KAIROS')` | `Feature("KAIROS")` is **always true** (see `commands/featuregates/gates.go`). Sub-flags (`KAIROS_CHANNELS`, `KAIROS_GITHUB_WEBHOOKS`, …) still use `FEATURE_<name>` env when mirrored. |
| `kairosActive` / daily memory | `getKairosActive()` in memdir | `GouDemoSystemOpts.KairosActive`; **`ApplyGouDemoRuntimeEnv`** sets it from **`CLAUDE_CODE_GO_KAIROS_ACTIVE`** (default **on** when the variable is unset). |
| Cron tools (`CronCreate` / `CronDelete` / `CronList`) | Extra GrowthBook gates (`tengu_kairos_cron`, durable flags, …) | **`toolpool`**: enabled when **`CLAUDE_CODE_DISABLE_CRON`** is not truthy (`tool_enabled.go`). No GrowthBook in standalone CLI. |
| Transport tools (`SendUserFile`, `PushNotification`, `SubscribePR`, `SuggestBackgroundPR`) | Need assistant / webhook runtime | **Error-shaped responses only** in `tools/optional_tools.go` (wire parity); no real send/push/webhook without a bridge. |
| Permission legacy alias `Brief` | Maps to `SendUserMessage` | Same via `permissionrules.NormalizeLegacyToolName`. |

## MCP JSON (optional, any path)

MCP command/tool snapshots are **not** kept in this directory. Point **`GOU_DEMO_MCP_COMMANDS_JSON`** / **`GOU_DEMO_MCP_TOOLS_JSON`** (or gou-demo flags) at any JSON file you generate; see [`goc/mcpcommands`](../../mcpcommands/load.go) and [`goc/mcpcommands/testdata`](../../mcpcommands/testdata) for examples.
