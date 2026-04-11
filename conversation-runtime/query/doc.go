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
//   - TS-parity streaming (Anthropic SSE + [streamingtool.StreamingToolExecutor] + [toolexecution.RunToolUseChan]):
//     when [QueryParams.StreamingParity] and [StreamingParityPathEnabled]([BuildQueryConfig]) (env
//     GOU_QUERY_STREAMING_PARITY or GOU_DEMO_STREAMING_TOOL_EXECUTION) are true, [queryLoop] calls
//     [runStreamingParityModelLoop] instead of [QueryDeps.CallModel], using [goc/anthropicmessages.PostStream]
//     (or [QueryDeps.StreamPost] for tests), optional [goc/anthropicmessages.BetasForToolsJSON] on the request,
//     and [RunToolUseToolRunner] with [QueryDeps.ToolexecutionDeps].
//   - [QueryParams.CanUseTool] is [toolexecution.QueryCanUseToolFn] ([toolexecution.PermissionDecision] + error); [NewStreamingToolExecutor]
//     receives it so [streamingtool.StreamingToolExecutor]'s canUseTool path matches TS wiring.
//   - After [runAutocompact], [applyAutocompactSideEffects] applies [AutocompactResult.UpdatedTracking] /
//     [AutocompactResult.UpdatedContentReplacementState] onto [State] (in-memory; [QueryParams] is not mutated).
//
// Non-streaming agentic multi-round tool loops remain in goc/ccb-engine/localturn ([engine.Session.RunTurn]);
// the streaming parity path runs its own multi-round loop over [anthropicmessages.PostStream].
package query
