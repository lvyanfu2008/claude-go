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
//   - Hook execution defaults to no-op (PreCompact/PostCompact/SessionStart); host wires a real runner when
//     the Go hooks subsystem lands parity for non-tool hooks.
//   - Post-compact attachment regeneration defaults to empty; host wires concrete providers (file re-read,
//     plan/plan_mode, skills, agent listing, MCP, deferred tools) as those subsystems land parity.
//   - Telemetry (tengu_compact, tengu_auto_compact_succeeded, tengu_compact_ptl_retry) defaults to a logger
//     hook (see Deps.LogEvent); hosts wire their analytics.
//
// See .cursor/rules/claude-go-mirror-typescript.mdc. TS is source of truth; any deviation is called out inline.
package compactservice
