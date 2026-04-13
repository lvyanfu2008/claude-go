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

## Standalone layout note

If you maintain a **standalone** copy without a sibling `claude-code` tree, see [`README.STANDALONE.txt`](README.STANDALONE.txt) and run exports from your TS checkout, then copy `tools_api.json` into `commands/data/` as above.
