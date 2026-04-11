// Package query mirrors the exported surface of src/conversation-runtime/query.ts
// (QueryParams, query(), Continue/Terminal, State, QueryConfig, small helpers).
//
// Implemented toward TS parity:
//   - [PrependUserContext] / [AppendSystemContext] (src/utils/api.ts),
//   - [MessagesForQuery]: compact-boundary slice only; [runApplyToolResultBudget] then applies JSON
//     replacements by default (or [QueryDeps.ApplyToolResultBudget] when set),
//   - When [QueryDeps.CallModel] is set, [queryLoop] runs (TS order): [MessagesForQuery],
//     optional [QueryDeps.ApplyToolResultBudget], optional [QueryDeps.SnipCompact] (may yield boundary),
//     [QueryDeps.Microcompact], [QueryDeps.Autocompact] (receives snip token delta when set),
//     [PrependUserContext], then CallModel.
//   - [LocalTurnCallModel] + [ProductionDepsWithLocalTurn] wire [localturn.QueryBridgeRun] so
//     each protocol [localturn.StreamEvent] is JSON-marshaled into [QueryYield.StreamEvent].
//   - After [runAutocompact], [applyAutocompactSideEffects] applies [AutocompactResult.UpdatedTracking] /
//     [AutocompactResult.UpdatedContentReplacementState] onto [State] (in-memory; [QueryParams] is not mutated).
//
// The full agentic while-loop (compaction, snip, tool orchestration, max_turns between
// recursions, …) is still in TypeScript query.ts / goc/ccb-engine/localturn.
package query
