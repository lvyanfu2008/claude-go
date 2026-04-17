# `goc/commands/data/` (embed / tooling snapshots)

Built-in **`COMMANDS()`**, bundled skills, and builtin-plugin skills **do not live here** — listing metadata is authoritative in **`goc/commands/handwritten`** (see `handwritten_load.go`). Drift-check JSON is under **`../testdata/`** (`bun run export:builtin-commands`, `export:bundled-skills`, `export:builtin-plugin-skills`), then `cd goc && go run ./cmd/gencode-handwritten` after updating `builtin_commands_default.json`.

Additional prompt/skill rows can live under **`../builtin_*/`** (see [`../builtin_overlay/README.md`](../builtin_overlay/README.md)); `loadBuiltinCommands` merges them after the handwritten table (earlier names win on duplicate).

## Embedded

- **`tools_api.json`** — embedded via [`../tools_api_embed.go`](../tools_api_embed.go) (`//go:embed`). Source of truth is **`claude-code`** (`toolToAPISchema` / Zod). Workflow:
  1. In **`claude-code`**: `bun run export:tools-registry` → writes `claude-code/data/exports/commands/data/tools_api.json`.
  2. **Copy** that file over this path:  
     `cp /path/to/claude-code/data/exports/commands/data/tools_api.json /path/to/claude-go/commands/data/tools_api.json`  
     (Monorepo sibling example: from `claude-code` repo root,  
     `cp data/exports/commands/data/tools_api.json ../claude-go/commands/data/tools_api.json`.)
  3. In **`claude-go`**: `go test ./...`  
  See also the top-level [`../../README.md`](../../README.md) in this module.

- **Channel / AskUserQuestion parity**: For TS parity when channel relay is active, set **`CLAUDE_CODE_GO_ALLOWED_CHANNELS`** (non-empty, comma-separated) together with **`FEATURE_KAIROS`** or **`FEATURE_KAIROS_CHANNELS`** so `toolpool.GetTools` omits **AskUserQuestion**, matching `AskUserQuestionTool.isEnabled()` in TS.

## MCP JSON (optional, any path)

MCP command/tool snapshots are **not** kept in this directory. Point **`GOU_DEMO_MCP_COMMANDS_JSON`** / **`GOU_DEMO_MCP_TOOLS_JSON`** (or gou-demo flags) at any JSON file you generate; see [`goc/mcpcommands`](../../mcpcommands/load.go) and [`goc/mcpcommands/testdata`](../../mcpcommands/testdata) for examples.
