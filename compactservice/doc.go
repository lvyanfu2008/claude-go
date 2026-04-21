// Package compactservice is the TypeScript-parity port of
// claude-code/src/services/compact/.
//
// Surface parity targets (in TS src/services/compact/):
//   - compactConversation                → CompactConversation
//   - partialCompactConversation         → PartialCompactConversation (PLANNED)
//   - autoCompactIfNeeded                → AutoCompactIfNeeded
//   - stripImagesFromMessages            → StripImagesFromMessages
//   - stripReinjectedAttachments         → StripReinjectedAttachments
//   - truncateHeadForPTLRetry            → TruncateHeadForPTLRetry
//   - groupMessagesByApiRound            → GroupMessagesByApiRound
//   - buildPostCompactMessages           → BuildPostCompactMessages
//   - mergeHookInstructions              → MergeHookInstructions
//   - annotateBoundaryWithPreservedSegment → AnnotateBoundaryWithPreservedSegment
//   - getCompactPrompt / getCompactUserSummaryMessage / getPartialCompactPrompt / formatCompactSummary (prompt.ts)
//
// Injection points (mirror TS function-granular deps so hosts can substitute):
//   - Deps.StreamSummary       → streamCompactSummary (TS)
//   - Deps.PreCompactHooks     → executePreCompactHooks (TS)
//   - Deps.PostCompactHooks    → executePostCompactHooks (TS)
//   - Deps.SessionStartHooks   → processSessionStartHooks (TS)
//   - Deps.PostCompactAttachments → createPostCompactFileAttachments + helpers (TS)
//
// Intentional simplifications from TS (documented and TODO-flagged in code):
//   - runForkedAgent (cache-prefix sharing) is not ported; fallback streaming path is the only route.
//   - PreCompact/PostCompact/SessionStartHooks default to no-op inside compactservice.resolve unless the host sets Deps.
//   - conversation-runtime’s autocompact adapter wires goc/hookexec PreCompact, PostCompact, and SessionStart
//     runners (merged settings command hooks) for parity with TS execute*Compact / processSessionStartHooks.
//   - Post-compact attachment regeneration defaults to empty; host wires concrete providers (file re-read,
//     plan/plan_mode, skills, agent listing, MCP, deferred tools) as those subsystems land parity.
//   - Telemetry (tengu_compact, tengu_auto_compact_succeeded, tengu_compact_ptl_retry) defaults to a logger
//     hook (see Deps.LogEvent); hosts wire their analytics.
//
// See .cursor/rules/claude-go-mirror-typescript.mdc. TS is source of truth; any deviation is called out inline.
//
// Auto-compact context cap (optional): GOC_AUTOCOMPACT_MAX_CONTEXT_WINDOW and CLAUDE_CODE_AUTO_COMPACT_WINDOW
// (positive token counts) lower the model context window used for threshold math; both may be set — the tighter cap applies.
// OpenAI-compatible `max_tokens` (query parity + autocompact) is clamped in [goc/conversation-runtime/query]
// like TS: CLAUDE_CODE_OPENAI_MAX_OUTPUT_TOKENS_CAP (default 8192); legacy alias GOC_AUTOCOMPACT_OPENAI_MAX_COMPLETION_TOKENS
// when the TS-named env is unset.
package compactservice
