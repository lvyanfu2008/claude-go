# claude-go (`goc`)

Go mirror of the Claude Code / engine paths: `ccb-engine`, `toolexecution`, `gou-demo`, etc. TypeScript in **`claude-code`** remains the semantic source for many behaviors; this module locks parity with tests and embedded snapshots.

## `tools_api.json` (tool `input_schema` / Zod → API)

Embedded at [`commands/data/tools_api.json`](commands/data/tools_api.json) via [`commands/tools_api_embed.go`](commands/tools_api_embed.go). Built-in tool shapes used by `anthropic.mustExportInputSchema` / `InputSchemaFromTSAPIExport` must match this file.

**Regenerate from `claude-code` and install into `claude-go`:**

```bash
# In claude-code (sibling of claude-go under the same parent directory)
cd ../claude-code
bun run export:tools-registry

# Copy export into this repo (embed path)
cp data/exports/commands/data/tools_api.json ../claude-go/commands/data/tools_api.json
```

Then refresh Zod parity goldens (expects the same sibling layout):

```bash
cd ../claude-code
bun run zod-parity-goldens
```

**Verify:**

```bash
cd ../claude-go
go test ./...
```

More detail on embeds and channel flags: [`commands/data/README.md`](commands/data/README.md).

## `GO_TOOL_INPUT_VALIDATOR=zog` (extra **`BashZog`** tool)

When **`GO_TOOL_INPUT_VALIDATOR=zog`** (see [`internal/toolvalidator/mode.go`](internal/toolvalidator/mode.go)):

- The usual **`Bash`** tool row still comes from embed **`tools_api.json`** (JSON Schema + [`toolrefine`](internal/toolrefine)) like other tools.
- A second tool **`BashZog`** is added (same shell execution as `Bash` in the Go runner). Its **`description` / `input_schema`** come from the checked-in snapshot [`ccb-engine/bashzog/bash_tool.json`](ccb-engine/bashzog/bash_tool.json); **Zog** validates **`BashZog`** input in [`zoglayer`](internal/zoglayer) plus `toolrefine`, aligned with the TS `fullInputSchema` **wire** (including optional `_simulatedSedEdit`). The snapshot stays model-facing on purpose and can omit internal-only keys.
- **[`toolpool.AssembleToolPoolFromEmbedded`](toolpool/assemble_embedded.go)** appends the **`BashZog`** [`types.ToolSpec`](types/tool.go) after the built-in list; **[`anthropic.GouParityToolList`](ccb-engine/internal/anthropic/canonical_demo_tools.go)** appends the matching tool definition for tests and parity runners.
- **Diff / parity:** checked-in [`ccb-engine/bashzog/bash_zog_tool_export.json`](ccb-engine/bashzog/bash_zog_tool_export.json) is the API-shaped **`BashZog`** row (pretty JSON). Regenerate from `claude-go`: `go run ./cmd/export-bashzog-json` (or `-stdout` to pipe to `diff` / `jq`).

Default (unset or any value other than `zog`) omits **`BashZog`** and keeps only **`Bash`** from the export pipeline.

## Standalone layout note

If you maintain a **standalone** copy without a sibling `claude-code` tree, see [`README.STANDALONE.txt`](README.STANDALONE.txt) and run exports from your TS checkout, then copy `tools_api.json` into `commands/data/` as above.
