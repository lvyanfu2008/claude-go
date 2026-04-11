package messagesapi

// Options replaces TS feature gates and Statsig checks for the Go API-normalize path.
type Options struct {
	// ToolSearchEnabled mirrors isToolSearchEnabledOptimistic().
	ToolSearchEnabled bool
	// ToolrefDeferJ8m mirrors tengu_toolref_defer_j8m (relocateToolReferenceSiblings, skip boundary inject).
	ToolrefDeferJ8m bool
	// ChairSermon mirrors tengu_chair_sermon (smooshSystemReminderSiblings, mergeUserContentBlocks universal smoosh).
	ChairSermon bool
	// HistorySnip mirrors feature('HISTORY_SNIP') && isSnipRuntimeEnabled (append [id:] tags).
	HistorySnip bool
	// TestMode mirrors NODE_ENV==='test' (skip snip tags).
	TestMode bool
	// NonInteractive mirrors getIsNonInteractiveSession() for user-facing attachment error strings.
	NonInteractive bool
	// EnableTaskReminder mirrors isTodoV2Enabled() for attachment type task_reminder.
	EnableTaskReminder bool
	// SkipImageValidation skips validateImagesForAPI when true.
	SkipImageValidation bool
	// VerifyPlanToolEnabled mirrors CLAUDE_CODE_VERIFY_PLAN === 'true' for verify_plan_reminder attachment.
	VerifyPlanToolEnabled bool
	// AgentSwarmsEnabled mirrors isAgentSwarmsEnabled() for teammate_mailbox / team_context attachments.
	AgentSwarmsEnabled bool
	// ExperimentalSkillSearch mirrors feature('EXPERIMENTAL_SKILL_SEARCH') for skill_discovery attachment.
	ExperimentalSkillSearch bool

	// PlanModeInterviewPhase mirrors isPlanModeInterviewPhaseEnabled(). When true, non-sparse plan_mode
	// uses getPlanModeInterviewInstructions; sparse reminder uses the iterative workflow sentence.
	PlanModeInterviewPhase bool
	// PlanPhase4Variant mirrors getPewterLedgerVariant: "", "trim", "cut", or "cap" (5-phase Phase 4 only).
	PlanPhase4Variant string
	// PlanModeV2AgentCount max Plan agents in Phase 2 (1–10); 0 means default 1.
	PlanModeV2AgentCount int
	// PlanModeV2ExploreAgentCount max Explore agents in Phase 1 (1–10); 0 means default 3.
	PlanModeV2ExploreAgentCount int
	// ExplorePlanAgentsEnabled mirrors areExplorePlanAgentsEnabled() (interview-path Explore bullet).
	ExplorePlanAgentsEnabled bool
	// PlanModeEmbeddedSearchTools mirrors hasEmbeddedSearchTools() for interview read-only tool list.
	PlanModeEmbeddedSearchTools bool

	// CompactAllTextUserContent when true, collapses every all-text user row (see collapseAllTextUserContentBlocks).
	// Default false matches TS normalizeMessagesForAPI: joinTextAtSeam keeps sibling text blocks (newline on the text-text seam only);
	// mergeUserContentBlocks appends attachment blocks without folding all text into one block.
	CompactAllTextUserContent bool
}

// DefaultOptions matches typical CLI defaults (most gates off).
func DefaultOptions() Options {
	return Options{}
}
