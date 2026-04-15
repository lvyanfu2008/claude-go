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
//   - TS-parity streaming ([streamingtool.StreamingToolExecutor] + [toolexecution.RunToolUseChan]):
//     when [QueryParams.StreamingParity] and [StreamingParityPathEnabled]([BuildQueryConfig]) (env
//     GOU_QUERY_STREAMING_PARITY or GOU_DEMO_STREAMING_TOOL_EXECUTION) are true, [queryLoop] calls either
//     [runStreamingParityModelLoop] (Anthropic Messages API + [goc/anthropicmessages.PostStream] or
//     [QueryDeps.StreamPost]) or, when [StreamingUsesOpenAIChat] is true (TS modelType openai /
//     CLAUDE_CODE_USE_OPENAI), [runOpenAIStreamingParityModelLoop] mirroring TS [queryModelOpenAI]:
//     POST /v1/chat/completions with stream:true, OPENAI_API_KEY + OPENAI_BASE_URL, wire conversion
//     matching src/api-client/openai, SSE adapted via [openAIStreamAdapter] to the same
//     [assistantStreamAccumulator] events. Tests set GOU_QUERY_STREAMING_FORCE_ANTHROPIC=1 ([TestMain])
//     so injected Anthropic [QueryDeps.StreamPost] is used unless a test clears it.
//     Optional [QueryDeps.OpenAIPostStream] overrides [PostOpenAIChatStream] for OpenAI streaming tests.
//     When [OpenAIChatNoStreamEnabled] (GOU_QUERY_OPENAI_CHAT_NO_STREAM), OpenAI parity uses
//     [runOpenAINonStreamingParityModelLoop]: one non-streaming JSON response per round, replayed
//     through [ReplayOpenAINonStreamChatResponse] into the same accumulator path (no SSE).
//   - Debug: GOU_QUERY_LOG_USER_CONTEXT=1 logs [QueryParams.UserContext] JSON to stderr before [PrependUserContext]
//     (see [LogQueryUserContextIfEnabled]). GOU_QUERY_LOG_OPENAI_NONSTREAM_WORK=1 logs the initial work slice
//     JSON via [ccb-engine/diaglog.Line] at the start of [runOpenAINonStreamingParityModelLoop] (truncated at 32KiB).
//   - [QueryParams.CanUseTool] is [toolexecution.QueryCanUseToolFn] ([toolexecution.PermissionDecision] + error); [NewStreamingToolExecutor]
//     receives it so [streamingtool.StreamingToolExecutor]'s canUseTool path matches TS wiring.
//   - After [runAutocompact], [applyAutocompactSideEffects] applies [AutocompactResult.UpdatedTracking] /
//     [AutocompactResult.UpdatedContentReplacementState] onto [State] (in-memory; [QueryParams] is not mutated).
//
// Multi-round tool loops for the streaming parity path run over [anthropicmessages.PostStream] (or OpenAI chat stream when configured).
package query
