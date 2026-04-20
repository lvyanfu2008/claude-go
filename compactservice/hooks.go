package compactservice

import "context"

// PreCompactHookInput mirrors the TS input object to executePreCompactHooks.
type PreCompactHookInput struct {
	Trigger            CompactTrigger
	CustomInstructions string
}

// PreCompactHookResult mirrors { newCustomInstructions, userDisplayMessage } in TS.
// When Blocked is true, the compaction is aborted without error (TS blockingError behavior).
type PreCompactHookResult struct {
	NewCustomInstructions string
	UserDisplayMessage    string
	// Blocked when true signals that pre-compact hooks want to abort compaction
	// without treating it as an error (TS blockingError pattern). The caller should
	// return early with WasCompacted=false.
	Blocked bool
}

// PreCompactHookRunner is the injection point for PreCompact hooks (TS executePreCompactHooks).
// Default implementation is no-op; hosts wire a real runner when Go ports the non-tool hooks subsystem.
type PreCompactHookRunner func(ctx context.Context, input PreCompactHookInput) (PreCompactHookResult, error)

// PostCompactHookInput mirrors the TS executePostCompactHooks input.
type PostCompactHookInput struct {
	Trigger        CompactTrigger
	CompactSummary string
}

// PostCompactHookResult mirrors TS output.
type PostCompactHookResult struct {
	UserDisplayMessage string
}

// PostCompactHookRunner mirrors TS executePostCompactHooks.
type PostCompactHookRunner func(ctx context.Context, input PostCompactHookInput) (PostCompactHookResult, error)

// SessionStartHookTrigger mirrors the TS argument to processSessionStartHooks ('compact' | 'startup' | ...).
type SessionStartHookTrigger string

const (
	SessionStartTriggerCompact SessionStartHookTrigger = "compact"
	SessionStartTriggerStartup SessionStartHookTrigger = "startup"
)

// SessionStartHookInput mirrors the second TS arg { model }.
type SessionStartHookInput struct {
	Model string
}

// SessionStartHookRunner mirrors processSessionStartHooks(trigger, { model }).
// Returns any hook-emitted attachment/system messages to append after compaction.
type SessionStartHookRunner func(ctx context.Context, trigger SessionStartHookTrigger, in SessionStartHookInput) ([]HookResultMessage, error)

// NoopPreCompactHookRunner is the safe default when no host runner is wired.
func NoopPreCompactHookRunner(_ context.Context, _ PreCompactHookInput) (PreCompactHookResult, error) {
	return PreCompactHookResult{}, nil
}

// NoopPostCompactHookRunner is the safe default when no host runner is wired.
func NoopPostCompactHookRunner(_ context.Context, _ PostCompactHookInput) (PostCompactHookResult, error) {
	return PostCompactHookResult{}, nil
}

// NoopSessionStartHookRunner is the safe default when no host runner is wired.
func NoopSessionStartHookRunner(_ context.Context, _ SessionStartHookTrigger, _ SessionStartHookInput) ([]HookResultMessage, error) {
	return nil, nil
}
