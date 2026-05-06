// Package toolresultpersist mirrors src/utils/toolResultStorage.ts: persist large tool results
// to {projectDir}/{sessionId}/tool-results/ and enforce per-message aggregate size budgets.
package toolresultpersist

// Subdirectory name for tool results within a session directory.
const ToolResultsSubdir = "tool-results"

// XML tags wrapping persisted output messages in tool_result content.
const (
	PersistedOutputTag        = "<persisted-output>"
	PersistedOutputClosingTag = "</persisted-output>"
)

// Message used when tool result content was cleared without persisting to file.
const ToolResultClearedMessage = "[Old tool result content cleared]"

// Preview size in bytes for the reference message shown to the model.
const PreviewSizeBytes = 2000

// Token estimation constant (TS: BYTES_PER_TOKEN = 4).
const BytesPerToken = 4

// Default max characters for tool results before persistence (TS: DEFAULT_MAX_RESULT_SIZE_CHARS = 50_000).
const DefaultMaxResultSizeChars = 50_000

// Max tool result tokens (TS: MAX_TOOL_RESULT_TOKENS = 100_000).
const MaxToolResultTokens = 100_000

// Max tool result bytes = tokens * bytes_per_token = 400_000 (TS: MAX_TOOL_RESULT_BYTES).
const MaxToolResultBytes = MaxToolResultTokens * BytesPerToken

// Default max aggregate characters for tool_result blocks within a single user message.
// (TS: MAX_TOOL_RESULTS_PER_MESSAGE_CHARS = 200_000).
const MaxToolResultsPerMessageChars = 200_000

// GrowthBook flag names for threshold overrides (TS parity; host-provided in Go).
const (
	FlagPersistThresholdOverride = "tengu_satin_quoll"
	FlagPerMessageBudgetOverride = "tengu_hawthorn_window"
	FlagEnforcementEnabled       = "tengu_hawthorn_steeple"
)

// Empty tool result marker mirrors TS inc-4586: inject a short marker so the model always
// has something to react to after tools that produce empty/no output.
const EmptyToolResultMarker = "(%s completed with no output)"
