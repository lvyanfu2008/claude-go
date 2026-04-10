Standalone copy of claude-code goc/ (module goc).

Upstream workflow: open the claude-code repo and read:
  docs/plans/goc-standalone-extraction.md

Regenerate embedded JSON from TS repo:
  bun run export:tools-registry
  (and other export:* scripts listed in that doc)

Then: cd "claude-go" && go test ./...
