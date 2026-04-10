// This file mirrors data shapes produced by src/context.ts (getSystemContext, getUserContext,
// and related git/CLAUDE.md injection). Logic stays in TS; Go uses these structs for parity tests / bridges.
package types

// MaxGitStatusChars matches src/context.ts MAX_STATUS_CHARS (git status truncation before append to prompt).
const MaxGitStatusChars = 2000

// SystemContext is the object shape returned by getSystemContext() in src/context.ts.
// Keys are optional fragments merged into the system prompt (except when skipped by env/feature).
type SystemContext struct {
	// GitStatus is the full multi-line snapshot from getGitStatus() (branch, main branch, user, status, recent commits).
	GitStatus *string `json:"gitStatus,omitempty"`
	// CacheBreaker is set when feature BREAK_CACHE_COMMAND and getSystemPromptInjection() are set (ant-only cache bust).
	CacheBreaker *string `json:"cacheBreaker,omitempty"`
}

// UserContext is the object shape returned by getUserContext() in src/context.ts.
type UserContext struct {
	// ClaudeMd is aggregated AGENTS.md / memory file content when discovery is enabled.
	ClaudeMd *string `json:"claudeMd,omitempty"`
	// CurrentDate is always set in TS: "Today's date is <ISO local date>."
	CurrentDate string `json:"currentDate"`
}

// TS module state: let systemPromptInjection: string | null (getSystemPromptInjection / setSystemPromptInjection).
// No extra Go type — use *string where a bridge stores the same value.
