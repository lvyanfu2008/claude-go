# /batch

Research and plan a large-scale change, then execute it in parallel across 5-30 isolated worktree agents that each open a PR.

## When To Use

Use when the user wants to make a sweeping, mechanical change across many files (migrations, refactors, bulk renames) that can be decomposed into independent parallel units.

## Arguments

`<instruction>`

## Usage Guardrails

- If no instruction is provided, ask for one and show examples.
- Only run in a git repository because this flow depends on isolated worktrees and PR creation.

## Worker Instructions (Shared Template)

After a worker finishes implementing its assigned change:

1. **Simplify**: run the `simplify` skill to review and clean up changes.
2. **Run unit tests**: execute the repository's normal test command(s) and fix failures.
3. **Run e2e verification**: follow the coordinator-provided e2e recipe, or skip only when explicitly allowed.
4. **Commit and push**: create a clear commit, push branch, open PR if possible.
5. **Report**: end with one line:
   - `PR: <url>`
   - or `PR: none - <reason>`

## Coordinator Flow

### Phase 1: Research And Plan (Plan Mode)

1. Enter plan mode.
2. Research impacted files, patterns, and conventions.
3. Split work into **5-30 independent units** that:
   - can be implemented in isolated worktrees,
   - can merge independently,
   - are roughly similar in size.
4. Define concrete e2e test recipe per unit category (UI, API, CLI, or integration test path).
5. If no clear e2e path exists, ask the user to pick one.
6. Produce a plan containing:
   - research summary,
   - numbered work units (title, files, one-line change),
   - e2e recipe,
   - exact worker instruction template.
7. Exit plan mode and request approval.

### Phase 2: Spawn Workers (After Approval)

1. Launch one background worktree agent per work unit.
2. Include full context in each worker prompt:
   - global objective,
   - that unit's files and scope,
   - relevant conventions,
   - e2e recipe,
   - shared worker template above.
3. Prefer general-purpose worker unless a specialized worker is clearly better.

### Phase 3: Track Progress

1. Render a status table after launch.
2. As workers finish, parse the final `PR:` line and update status (`done` or `failed`).
3. Show final completion summary when all workers report.

## Usage Examples

- `/batch migrate from react to vue`
- `/batch replace all uses of lodash with native equivalents`
- `/batch add type annotations to all untyped function parameters`
