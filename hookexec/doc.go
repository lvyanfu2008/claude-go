// Package hookexec ports the non-REPL **command-hook** slice of Claude Code’s hooks subsystem toward
// parity with src/utils/hooks.ts (getMatchingHooks matchQuery switch, matchesPattern filtering semantics,
// executeHooksOutsideREPL parallel command execution).
//
// Layers (incremental parity):
//  1) Settings merge + matcher + matchQuery derivation + parallel command runs — implemented.
//  2) PreCompact / PostCompact / SessionStart / InstructionsLoaded / UserPromptSubmit command hooks — implemented (see compact_hooks.go, user_prompt_submit.go).
//  3) Policy / trust gates — partial (env CLAUDE_CODE_POLICY_DISABLE_ALL_HOOKS; interactive trust stub).
//  4) Plugin + session snapshot hooks, prompt/agent/http/callback/function — not ported (TS-only or REPL).
//  5) Per-event stdin validation (Zod) — callers must build JSON matching coreSchemas HookInput union.
//
// Merged hook tables: user ~/.claude/settings.json, project .claude/settings.go.json, .claude/settings.local.json.
package hookexec
