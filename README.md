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

## `GO_TOOL_INPUT_VALIDATOR=zog` (Bash Go-sourced path)

When **`GO_TOOL_INPUT_VALIDATOR=zog`** (see [`internal/toolvalidator/mode.go`](internal/toolvalidator/mode.go)):

- **Bash** validation uses **Zog** plus [`toolrefine`](internal/toolrefine) for tools registered in [`zoglayer`](internal/zoglayer); **`toolvalidator.ValidateInput` does not require an `input_schema` from `tools_api.json` for those tools** (schema may be nil).
- **Bash** model-facing **`name` / `description` / `input_schema`** come from the checked-in snapshot [`ccb-engine/bashzog/bash_tool.json`](ccb-engine/bashzog/bash_tool.json) (via [`ccb-engine/bashzog`](ccb-engine/bashzog)), not from a runtime read of embed `tools_api.json`. Refresh that JSON when the TS export for Bash changes and you want Go to stay aligned.
- **[`toolpool.AssembleToolPoolFromEmbedded`](toolpool/assemble_embedded.go)** and **[`anthropic.bashToolDefinition`](ccb-engine/internal/anthropic/canonical_demo_tools.go)** substitute the Bash row when this mode is on. Other tools still use the embed / export pipeline.

Default (unset or any value other than `zog`) keeps the **JSON Schema + embed `tools_api.json`** path for Bash like all other tools.

## Standalone layout note

If you maintain a **standalone** copy without a sibling `claude-code` tree, see [`README.STANDALONE.txt`](README.STANDALONE.txt) and run exports from your TS checkout, then copy `tools_api.json` into `commands/data/` as above.
