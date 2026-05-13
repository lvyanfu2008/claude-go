# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Hard constraint

**No TypeScript runtime dependency**: builds/tests must not spawn Bun, Node, or `claude-code` `.ts` entrypoints. Embedded JSON/markdown from the TS codebase are static assets only.

## Commands

```bash
go test ./...
go vet ./...
go build -o /dev/null ./cmd/claude
golangci-lint run ./...
```

## go:generate

After syncing TS JSON exports (out-of-band, not CI), regenerate:

```bash
go generate ./tools/toolparity
go generate ./commands/handwritten
```

## Key details

- Module name is `goc` (not `claude-go`).
- `canUseTool` / `QueryCanUseToolFn` returns `PermissionDecision` (`allow` / `deny` / `ask`) — 3-state, not a bool.
- Branch naming uses conventional prefixes (e.g., `feature/`, `fix/`, `base-`).

## Key env vars

| Var | Effect |
|-----|--------|
| `CLAUDE_CODE_ENABLE_TASKS=1` | Enable Todo v2 (TaskCreate/Get/List/Update) |
| `OPENAI_ENABLE_THINKING=1` | Enable thinking mode for OpenAI-compatible providers |
| `GO_TOOL_INPUT_VALIDATOR=zog` | Use zog-validated Bash (BashZog) |
| `CCB_ENGINE_MODEL` | Model ID (e.g., `deepseek-v4-pro`) |
