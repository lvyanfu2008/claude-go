package slashresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/types"
)

const (
	batchMinAgents = 5
	batchMaxAgents = 30
)

const batchWorkerInstructions = "After you finish implementing the change:\n" +
	"1. **Simplify** — Invoke the `Skill` tool with `skill: \"simplify\"` to review and clean up your changes.\n" +
	"2. **Run unit tests** — Run the project's test suite (check for package.json scripts, Makefile targets, or common commands like `npm test`, `bun test`, `pytest`, `go test`). If tests fail, fix them.\n" +
	"3. **Test end-to-end** — Follow the e2e test recipe from the coordinator's prompt (below). If the recipe says to skip e2e for this unit, skip it.\n" +
	"4. **Commit and push** — Commit all changes with a clear message, push the branch, and create a PR with `gh pr create`. Use a descriptive title. If `gh` is not available or the push fails, note it in your final message.\n" +
	"5. **Report** — End with a single line: `PR: <url>` so the coordinator can track it. If no PR was created, end with `PR: none — <reason>`."

const batchNotGitRepo = `This is not a git repository. The ` + "`/batch`" + ` command requires a git repo because it spawns agents in isolated git worktrees and creates PRs from each. Initialize a repo first, or run this from inside an existing one.`

const batchMissingInstruction = `Provide an instruction describing the batch change you want to make.

Examples:
  /batch migrate from react to vue
  /batch replace all uses of lodash with native equivalents
  /batch add type annotations to all untyped function parameters`

func buildBatchPrompt(instruction string) string {
	return fmt.Sprintf(`# Batch: Parallel Work Orchestration

You are orchestrating a large, parallelizable change across this codebase.

## User Instruction

%s

## Phase 1: Research and Plan (Plan Mode)

Call the `+"`EnterPlanMode`"+` tool now to enter plan mode, then:

1. **Understand the scope.** Launch one or more subagents (in the foreground — you need their results) to deeply research what this instruction touches. Find all the files, patterns, and call sites that need to change. Understand the existing conventions so the migration is consistent.

2. **Decompose into independent units.** Break the work into %d–%d self-contained units. Each unit must:
   - Be independently implementable in an isolated git worktree (no shared state with sibling units)
   - Be mergeable on its own without depending on another unit's PR landing first
   - Be roughly uniform in size (split large units, merge trivial ones)

   Scale the count to the actual work: few files → closer to %d; hundreds of files → closer to %d. Prefer per-directory or per-module slicing over arbitrary file lists.

3. **Determine the e2e test recipe.** Figure out how a worker can verify its change actually works end-to-end — not just that unit tests pass. Look for:
   - A `+"`claude-in-chrome`"+` skill or browser-automation tool (for UI changes: click through the affected flow, screenshot the result)
   - A `+"`tmux`"+` or CLI-verifier skill (for CLI changes: launch the app interactively, exercise the changed behavior)
   - A dev-server + curl pattern (for API changes: start the server, hit the affected endpoints)
   - An existing e2e/integration test suite the worker can run

   If you cannot find a concrete e2e path, use the `+"`AskUserQuestion`"+` tool to ask the user how to verify this change end-to-end. Offer 2–3 specific options based on what you found (e.g., "Screenshot via chrome extension", "Run `+"`bun run dev`"+` and curl the endpoint", "No e2e — unit tests are sufficient"). Do not skip this — the workers cannot ask the user themselves.

   Write the recipe as a short, concrete set of steps that a worker can execute autonomously. Include any setup (start a dev server, build first) and the exact command/interaction to verify.

4. **Write the plan.** In your plan file, include:
   - A summary of what you found during research
   - A numbered list of work units — for each: a short title, the list of files/directories it covers, and a one-line description of the change
   - The e2e test recipe (or "skip e2e because …" if the user chose that)
   - The exact worker instructions you will give each agent (the shared template)

5. Call `+"`ExitPlanMode`"+` to present the plan for approval.

## Phase 2: Spawn Workers (After Plan Approval)

Once the plan is approved, spawn one background agent per work unit using the `+"`Agent`"+` tool. **All agents must use `+"`isolation: \"worktree\"`"+` and `+"`run_in_background: true`"+`.** Launch them all in a single message block so they run in parallel.

For each agent, the prompt must be fully self-contained. Include:
- The overall goal (the user's instruction)
- This unit's specific task (title, file list, change description — copied verbatim from your plan)
- Any codebase conventions you discovered that the worker needs to follow
- The e2e test recipe from your plan (or "skip e2e because …")
- The worker instructions below, copied verbatim:

`+"```"+`
%s
`+"```"+`

Use `+"`subagent_type: \"general-purpose\"`"+` unless a more specific agent type fits.

## Phase 3: Track Progress

After launching all workers, render an initial status table:

| # | Unit | Status | PR |
|---|------|--------|----|
| 1 | <title> | running | — |
| 2 | <title> | running | — |

As background-agent completion notifications arrive, parse the `+"`PR: <url>`"+` line from each agent's result and re-render the table with updated status (`+"`done`"+` / `+"`failed`"+`) and PR links. Keep a brief failure note for any agent that did not produce a PR.

When all agents have reported, render the final table and a one-line summary (e.g., "22/24 units landed as PRs").
`, instruction, batchMinAgents, batchMaxAgents, batchMinAgents, batchMaxAgents, batchWorkerInstructions)
}

func isGitRepoAt(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	st, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return st.IsDir() || st.Mode()&os.ModeSymlink != 0
}

func resolveBatch(args string, cwd string) (types.SlashResolveResult, error) {
	instruction := strings.TrimSpace(args)
	if instruction == "" {
		return types.SlashResolveResult{UserText: batchMissingInstruction, Source: types.SlashResolveBundledEmbed}, nil
	}
	wd := cwd
	if wd == "" {
		wd, _ = os.Getwd()
	}
	if !isGitRepoAt(wd) {
		return types.SlashResolveResult{UserText: batchNotGitRepo, Source: types.SlashResolveBundledEmbed}, nil
	}
	return types.SlashResolveResult{UserText: buildBatchPrompt(instruction), Source: types.SlashResolveBundledEmbed}, nil
}

const (
	loopDefaultInterval = "10m"
	cronCreateTool      = "CronCreate"
	cronDeleteTool      = "CronDelete"
	defaultMaxAgeDays   = 30
)

const loopUsage = `Usage: /loop [interval] <prompt>

Run a prompt or slash command on a recurring interval.

Intervals: Ns, Nm, Nh, Nd (e.g. 5m, 30m, 2h, 1d). Minimum granularity is 1 minute.
If no interval is specified, defaults to ` + loopDefaultInterval + `.

Examples:
  /loop 5m /babysit-prs
  /loop 30m check the deploy
  /loop 1h /standup 1
  /loop check the deploy          (defaults to ` + loopDefaultInterval + `)
  /loop check the deploy every 20m`

func buildLoopPrompt(args string) string {
	return fmt.Sprintf(`# /loop — schedule a recurring prompt

Parse the input below into `+"`[interval] <prompt…>`"+` and schedule it with %s.

## Parsing (in priority order)

1. **Leading token**: if the first whitespace-delimited token matches `+"`^\\d+[smhd]$`"+` (e.g. `+"`5m`"+`, `+"`2h`"+`), that's the interval; the rest is the prompt.
2. **Trailing "every" clause**: otherwise, if the input ends with `+"`every <N><unit>`"+` or `+"`every <N> <unit-word>`"+` (e.g. `+"`every 20m`"+`, `+"`every 5 minutes`"+`, `+"`every 2 hours`"+`), extract that as the interval and strip it from the prompt. Only match when what follows "every" is a time expression — `+"`check every PR`"+` has no interval.
3. **Default**: otherwise, interval is `+"`%s`"+` and the entire input is the prompt.

If the resulting prompt is empty, show usage `+"`/loop [interval] <prompt>`"+` and stop — do not call %s.

Examples:
- `+"`5m /babysit-prs`"+` → interval `+"`5m`"+`, prompt `+"`/babysit-prs`"+` (rule 1)
- `+"`check the deploy every 20m`"+` → interval `+"`20m`"+`, prompt `+"`check the deploy`"+` (rule 2)
- `+"`run tests every 5 minutes`"+` → interval `+"`5m`"+`, prompt `+"`run tests`"+` (rule 2)
- `+"`check the deploy`"+` → interval `+"`%s`"+`, prompt `+"`check the deploy`"+` (rule 3)
- `+"`check every PR`"+` → interval `+"`%s`"+`, prompt `+"`check every PR`"+` (rule 3 — "every" not followed by time)
- `+"`5m`"+` → empty prompt → show usage

## Interval → cron

Supported suffixes: `+"`s`"+` (seconds, rounded up to nearest minute, min 1), `+"`m`"+` (minutes), `+"`h`"+` (hours), `+"`d`"+` (days). Convert:

| Interval pattern      | Cron expression     | Notes                                    |
|-----------------------|---------------------|------------------------------------------|
| `+"`Nm`"+` where N ≤ 59   | `+"`*/N * * * *`"+`     | every N minutes                          |
| `+"`Nm`"+` where N ≥ 60   | `+"`0 */H * * *`"+`     | round to hours (H = N/60, must divide 24)|
| `+"`Nh`"+` where N ≤ 23   | `+"`0 */N * * *`"+`     | every N hours                            |
| `+"`Nd`"+`                | `+"`0 0 */N * *`"+`     | every N days at midnight local           |
| `+"`Ns`"+`                | treat as `+"`ceil(N/60)m`"+` | cron minimum granularity is 1 minute  |

**If the interval doesn't cleanly divide its unit** (e.g. `+"`7m`"+` → `+"`*/7 * * * *`"+` gives uneven gaps at :56→:00; `+"`90m`"+` → 1.5h which cron can't express), pick the nearest clean interval and tell the user what you rounded to before scheduling.

## Action

1. Call %s with:
   - `+"`cron`"+`: the expression from the table above
   - `+"`prompt`"+`: the parsed prompt from above, verbatim (slash commands are passed through unchanged)
   - `+"`recurring`"+`: `+"`true`"+`
2. Briefly confirm: what's scheduled, the cron expression, the human-readable cadence, that recurring tasks auto-expire after %d days, and that they can cancel sooner with %s (include the job ID).
3. **Then immediately execute the parsed prompt now** — don't wait for the first cron fire. If it's a slash command, invoke it via the Skill tool; otherwise act on it directly.

## Input

%s`, cronCreateTool, loopDefaultInterval, cronCreateTool, loopDefaultInterval, loopDefaultInterval, cronCreateTool, defaultMaxAgeDays, cronDeleteTool, args)
}

func resolveLoop(args string) (types.SlashResolveResult, error) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return types.SlashResolveResult{UserText: loopUsage, Source: types.SlashResolveBundledEmbed}, nil
	}
	return types.SlashResolveResult{UserText: buildLoopPrompt(trimmed), Source: types.SlashResolveBundledEmbed}, nil
}
