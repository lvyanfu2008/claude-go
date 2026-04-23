package toolexecution

// Mirrors src/services/tools/toolExecution.ts exports and src/utils/messagesLiteralConstants.ts.

const (
	// HOOK_TIMING_DISPLAY_THRESHOLD_MS mirrors export in toolExecution.ts line ~135.
	HOOK_TIMING_DISPLAY_THRESHOLD_MS = 500
	// SLOW_PHASE_LOG_THRESHOLD_MS mirrors toolExecution.ts line ~138.
	SLOW_PHASE_LOG_THRESHOLD_MS = 2000
)

// MEMORY_CORRECTION_HINT mirrors messagesLiteralConstants.ts.
const MEMORY_CORRECTION_HINT = "\n\nNote: The user's next message may contain a correction or preference. Pay close attention — if they explain what went wrong or how they'd prefer you to work, consider saving that to memory for future sessions."

// CANCEL_MESSAGE mirrors messagesLiteralConstants.ts (toolExecution.ts abort path).
const CANCEL_MESSAGE = "The user doesn't want to take this action right now. STOP what you are doing and wait for the user to tell you how to proceed."

func withMemoryCorrectionHint(s string) string {
	return s + MEMORY_CORRECTION_HINT
}
