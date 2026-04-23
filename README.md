# claude-go (`goc`)

Go mirror of the Claude Code / engine paths: `ccb-engine`, `tools/toolexecution`, `gou-demo`, etc. TypeScript in **`claude-code`** remains the semantic source for many behaviors; this module locks parity with tests and embedded snapshots.

## `tools_api.json` (tool `input_schema` / Zod → API)

Embedded at [`commands/data/tools_api.json`](commands/data/tools_api.json) via [`commands/tools_api_embed.go`](commands/tools_api_embed.go) — the TypeScript `toolToAPISchema` / Zod export, kept in sync with [`toolpool` go wire](tools/toolpool/go_tool_wire.go) (`InputJSONSchema` from [`ToolSpecsFromGoWire`]). `anthropic.mustExportInputSchema` / `InputSchemaFromTSAPIExport` read **from that Go wire registry**, not from the JSON file at runtime; periodically **`bun run export:tools-registry`** in `claude-code` and copy here so the embed and native Go specs do not drift from TS.

**Regenerate from `claude-code` and install into `claude-go`:**

```bash
# In claude-code (sibling of claude-go under the same parent directory)
cd ../claude-code
bun run export:tools-registry

# Copy export into this repo (embed path)
cp data/exports/commands/data/tools_api.json ../claude-go/commands/data/tools_api.json
```

**Verify:**

```bash
cd ../claude-go
go test ./...
```

More detail on embeds and channel flags: [`commands/data/README.md`](commands/data/README.md).

## `GO_TOOL_INPUT_VALIDATOR=zog` (**`BashZog`** replaces **`Bash`**)

When **`GO_TOOL_INPUT_VALIDATOR=zog`** (see [`internal/toolvalidator/mode.go`](internal/toolvalidator/mode.go)):

- The built-in **`Bash`** row from embed **`tools_api.json`** is **removed** from the tool list and replaced by **`BashZog`** (same local shell execution in the Go runner). **`description`** is built by [`ccb-engine/bashzog/simple_prompt.go`](ccb-engine/bashzog/simple_prompt.go) **`GetSimplePrompt`** (port of `src/tools/BashTool/prompt.ts` **`getSimplePrompt`**). **`input_schema`** is defined in [`ccb-engine/bashzog/wire.go`](ccb-engine/bashzog/wire.go) (native map + marshal). Optional env: **`CLAUDE_CODE_GO_BASH_SANDBOX_PROMPT=1`** appends the “Command sandbox” section (without live SandboxManager JSON); **`CLAUDE_CODE_GO_ALLOW_UNSANDBOXED_BASH=1`** selects the unsandbox-override bullet branch; **`CLAUDE_CODE_GO_INTERNAL_MODEL_REPO=1`** turns off auto-undercover for `USER_TYPE=ant` (approximates internal repo). **Zog** validates **`BashZog`** in [`zoglayer`](internal/zoglayer) plus [`toolrefine`](internal/toolrefine), aligned with TS `fullInputSchema` (including optional `_simulatedSedEdit`).
- **[`toolpool.ReplaceBashToolSpecIfZogMode`](tools/toolpool/bash_zog.go)** (used by [`AssembleToolPoolFromEmbedded`](tools/toolpool/assemble_embedded.go)) swaps **`Bash` → `BashZog`** on the assembled pool. **[`ToolSpecsFromGoWire`](tools/toolpool/go_tool_wire.go)** / **`GetTools`** then expose **`BashZog`** instead of **`Bash`** in the same way as the main runtime list.
- **Ad-hoc export:** `go run ./cmd/export-bashzog-json` prints the API-shaped **`BashZog`** row to stdout (use `-out path` to write a file).

Default (unset or any value other than `zog`) keeps **`Bash`** in the go-wire pool and does not register **`BashZog`**.

## Standalone layout note

If you maintain a **standalone** copy without a sibling `claude-code` tree, see [`README.STANDALONE.txt`](README.STANDALONE.txt) and run exports from your TS checkout, then copy `tools_api.json` into `commands/data/` as above.
