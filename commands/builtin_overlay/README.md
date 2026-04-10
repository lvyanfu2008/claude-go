# Builtin command overlay (Go)

Place extra slash/skill definitions alongside the **handwritten** builtin table (`goc/commands/handwritten` + `z_jsondata.go`).

- This directory name **must** start with `builtin` (it is under `goc/commands/`).
- Supported files (recursive):
  - **`*.json`**: a JSON array of `types.Command`, or one command object.
  - **`*.md`**: SKILL-style YAML frontmatter + markdown body (same as `.claude/skills`). For `subdir/SKILL.md`, the command name is `subdir`. For other `*.md` files, the name is the file stem.
- **`README.md`** is ignored.
- If a command **name** already exists in the handwritten table, the disk entry is **skipped**.

Refresh the main table via `bun run export:builtin-commands` and `cd goc && go run ./cmd/gencode-handwritten` when changing TS `COMMANDS()`; use this tree for repo-local additions without regenerating.
