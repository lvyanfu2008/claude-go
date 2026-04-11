package query

// ContinueReason mirrors src/conversation-runtime/types/queryLoopTransitions.ts Continue.
type ContinueReason string

const (
	ContinueReasonCollapseDrainRetry      ContinueReason = "collapse_drain_retry"
	ContinueReasonReactiveCompactRetry    ContinueReason = "reactive_compact_retry"
	ContinueReasonMaxOutputTokensEscalate ContinueReason = "max_output_tokens_escalate"
	ContinueReasonMaxOutputTokensRecovery ContinueReason = "max_output_tokens_recovery"
	ContinueReasonStopHookBlocking        ContinueReason = "stop_hook_blocking"
	ContinueReasonTokenBudgetContinuation ContinueReason = "token_budget_continuation"
	ContinueReasonNextTurn                ContinueReason = "next_turn"
)

// Continue records why queryLoop continued to another iteration (TS Continue).
type Continue struct {
	Reason ContinueReason
	// Committed is set when Reason == ContinueReasonCollapseDrainRetry.
	Committed int
	// Attempt is set when Reason == ContinueReasonMaxOutputTokensRecovery.
	Attempt int
}

// TerminalReason mirrors src/conversation-runtime/types/queryLoopTransitions.ts Terminal.
type TerminalReason string

const (
	TerminalReasonBlockingLimit     TerminalReason = "blocking_limit"
	TerminalReasonImageError        TerminalReason = "image_error"
	TerminalReasonModelError        TerminalReason = "model_error"
	TerminalReasonAbortedStreaming  TerminalReason = "aborted_streaming"
	TerminalReasonPromptTooLong     TerminalReason = "prompt_too_long"
	TerminalReasonCompleted         TerminalReason = "completed"
	TerminalReasonStopHookPrevented TerminalReason = "stop_hook_prevented"
	TerminalReasonAbortedTools      TerminalReason = "aborted_tools"
	TerminalReasonHookStopped       TerminalReason = "hook_stopped"
	TerminalReasonMaxTurns          TerminalReason = "max_turns"
)

// Terminal is the queryLoop return value when the turn ends (TS Terminal).
type Terminal struct {
	Reason    TerminalReason
	Error     error // model_error
	TurnCount int   // max_turns
}
