// Package claudemd ports src/utils/claudemd.ts memory file discovery (getMemoryFiles) and
// getClaudeMds aggregation for Go-only hosts.
//
// Parity notes:
//   - getMemoryFiles-equivalent: [LoadMemoryFiles] is session-memoized; use [ClearMemoryFileCaches] or
//     [ResetMemoryFilesCache] to invalidate; env fingerprint in memory_files_cache.go keys the cache.
//   - InstructionsLoaded: runs from uncached [loadMemoryFilesUncached] (see goc/hookexec); one-shot
//     load_reason is session_start until [ResetMemoryFilesCache] (e.g. "compact" after autocompact).
//   - GrowthBook flags: CLAUDE_CODE_TENGU_MOTH_COPSE, CLAUDE_CODE_TENGU_PAPER_HALYARD.
//   - External @include: Force/Has on LoadOptions or CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED=1.
//   - Team memory: FEATURE_TEAMMEM, IsAutoMemoryEnabled, CLAUDE_CODE_TEAM_MEMORY_ENABLED (optional override).
//   - Hard opt-out: CLAUDE_CODE_DISABLE_USER_MEMORY, CLAUDE_CODE_DISABLE_PROJECT_MEMORY, CLAUDE_CODE_DISABLE_LOCAL_MEMORY
//     (applied on top of setting-source gates).
//
// Settings: project .claude/settings.json is TS-only. Go uses ~/.claude/settings.json (user),
// <cwd>/.claude/settings.go.json, and settings.local.json for env (settingsfile) and claudeMdExcludes (MergedClaudeMdExcludes).
//   - CLAUDE_CODE_SETTING_SOURCES: comma list user, project, local, flag, policy (default: all).
//     Use value "isolated" for allow-list empty → only policy+flag merge (SDK-style).
//   - User ~/.claude/settings.json or cowork_settings.json (CLAUDE_CODE_USE_COWORK_PLUGINS=1).
//   - CLAUDE_CODE_FLAG_SETTINGS_PATH for flagSettings file.
//   - Policy: managed-settings.json + managed-settings.d/*.json under ManagedFilePath().
//   - claudeMdExcludes arrays merge with concat + first-wins dedupe per lodash mergeArrays; matching uses
//     doublestar.Match on slash-normalized paths plus resolveExcludePatterns (realpath static prefix of absolute globs).
//
// Not ported: remote/MDM/HKCU policy chain beyond file-based managed settings; plugin settings base merge;
// claudeMdExcludes picomatch "dot: true" nuance (doublestar * semantics).
package claudemd
