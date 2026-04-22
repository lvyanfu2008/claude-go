package compactservice

import (
	"context"
	"os"
	"strings"
)

// Deps bundles the injection points for compactConversation + autoCompactIfNeeded.
// Mirrors the TS ad-hoc pattern of using ToolUseContext to reach each subsystem,
// but makes the surface explicit so hosts can substitute at function granularity
// without importing the full conversation-runtime.
type Deps struct {
	// Summarize performs the actual compaction LLM call. REQUIRED for real
	// compaction; tests can provide a synthetic summary.
	Summarize SummarizerFn

	// PreCompactHooks / PostCompactHooks / SessionStartHooks run the
	// corresponding hook phases. Nil is interpreted as the no-op runner.
	PreCompactHooks   PreCompactHookRunner
	PostCompactHooks  PostCompactHookRunner
	SessionStartHooks SessionStartHookRunner

	// PostCompactAttachments produces the attachments re-appended after the
	// summary (file re-read, plan, plan_mode, skills, agent listing, MCP,
	// deferred tools). Nil defaults to no-op.
	PostCompactAttachments PostCompactAttachmentProvider

	// Logger receives tengu_compact* events. Nil defaults to NoopLogger.
	Logger Logger

	// TranscriptPath is threaded into getCompactUserSummaryMessage so the
	// continuation prompt points the model at the full session log. May be empty.
	TranscriptPath string

	// NewUUID overrides the RFC-4122 v4 generator for tests.
	NewUUID func() string

	// Now overrides time.Now() for tests (returns RFC3339 nano-formatted string).
	Now func() string

	// ProactiveActive mirrors the proactive/isProactiveActive() branch in
	// getCompactUserSummaryMessage. Default false.
	ProactiveActive bool

	// AfterSuccessfulCompact runs after a successful compaction (TS runPostCompactCleanup
	// subset). Receives the same query source string as [CompactOptions.QuerySource] /
	// [RecompactionInfo.QuerySource] via [compactQuerySourceForCleanup]. Nil defaults to no-op.
	AfterSuccessfulCompact func(querySource string)
}

// resolve sets sensible defaults on Deps fields that are nil.
func (d *Deps) resolve() {
	if d.PreCompactHooks == nil {
		d.PreCompactHooks = NoopPreCompactHookRunner
	}
	if d.PostCompactHooks == nil {
		d.PostCompactHooks = NoopPostCompactHookRunner
	}
	if d.SessionStartHooks == nil {
		d.SessionStartHooks = NoopSessionStartHookRunner
	}
	if d.PostCompactAttachments == nil {
		d.PostCompactAttachments = NoopPostCompactAttachmentProvider
	}
	if d.Logger == nil {
		d.Logger = NoopLogger{}
	}
	if d.NewUUID == nil {
		d.NewUUID = newUUID
	}
	if d.Now == nil {
		d.Now = nowRFC3339
	}
	if d.AfterSuccessfulCompact == nil {
		d.AfterSuccessfulCompact = func(string) {}
	}
}

// --- environment-variable helpers (mirror src/utils/envUtils.ts isEnvTruthy) ---

// IsEnvTruthy matches TS isEnvTruthy: present + not "0"/"false"/empty-after-trim.
func IsEnvTruthy(name string) bool {
	v, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false":
		return false
	}
	return true
}

// --- context utility to match TS { signal: context.abortController.signal } ---

// contextIsCanceled reports whether ctx has been canceled (used for abort branches).
func contextIsCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
