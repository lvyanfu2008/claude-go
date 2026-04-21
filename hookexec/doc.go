// Package hookexec runs a subset of Claude Code settings hooks outside the TS REPL:
// InstructionsLoaded (fire-and-forget command hooks) and SessionStart (command hooks with
// hook_additional_context extraction). Matcher semantics mirror src/utils/hooks.ts matchesPattern
// for simple and pipe-separated patterns; merged hook tables follow settingsfile env merge paths
// (user ~/.claude/settings.json, project .claude/settings.go.json, .claude/settings.local.json).
package hookexec
