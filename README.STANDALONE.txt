Standalone copy of claude-code goc/ (module goc).

Upstream workflow: open the claude-code repo and read:
  docs/plans/goc-standalone-extraction.md

tools_api.json sync (full steps): see README.md in this directory.

Summary: in claude-code run `bun run export:tools-registry`, copy
  data/exports/commands/data/tools_api.json
  → claude-go/commands/data/tools_api.json
then optionally `bun run zod-parity-goldens` in claude-code, then `cd claude-go && go test ./...`

Other export:* scripts: listed in goc-standalone-extraction.md
