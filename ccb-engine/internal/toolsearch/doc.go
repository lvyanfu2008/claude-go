// Package toolsearch mirrors TS tool-search wiring in src/services/api/claude.ts and
// src/utils/toolSearch.ts for the ccb-engine HTTP path.
//
// Still intentionally narrower than full TS where the product depends on GrowthBook,
// token-counting APIs, or attachment pipelines not present in this Go host:
//   - tst-auto uses a char fallback (no token-count API). Measured size is scaled by
//     tstAutoDeferredCharScale (default 1.65, override CLAUDE_CODE_GO_TST_AUTO_CHAR_SCALE)
//     so static tools_api.json aligns better with TS tool.prompt() description sizes.
//   - Unsupported-model patterns beyond "haiku" are not read from Statsig.
//   - deferred_tools_delta / getDeferredToolsDelta attachments are not synthesized here;
//     use CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA=1 or USER_TYPE=ant to skip <available-deferred-tools> prepend only.
//   - isDeferredTool fork/Kairos/Brief branches are approximated via a static builtin map + mcp__ prefix.
//
// Gou-demo parity: when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1, tst-auto is treated as tst for wire tools[]
// (embedded export matches TS default defer; char-only tst-auto can otherwise leave deferred tools inline).
// CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT=0 disables only the <available-deferred-tools> prepend, not API tools[] filtering.
//
// Wire diagnostics: [LogWireRound] when CLAUDE_CODE_GO_TOOL_SEARCH_DIAG, CLAUDE_CODE_LOG_API_REQUEST_BODY, or GOU_DEMO_LOG is on (see wire_log.go); uses [diaglog.Line] (CCB_ENGINE_DIAG_TO_STDERR=1 for stderr).
//
// OpenAI /chat/completions (DeepSeek, etc.): TypeScript still applies the same filteredTools pass before
// queryModelOpenAI (src/services/api/claude.ts: filteredTools from useToolSearch + extractDiscoveredToolNames;
// openai/index.ts only strips defer_loading etc. in anthropicToolsToOpenAI — it does not re-expand the list).
// Go mirrors that by running [ApplyWire] for OpenAI compat too; [ExecToolSearchForRunner] mirrors
// ToolSearchTool.ts (select / keyword / mcp__ / scoring) and emits tool_result as a JSON array of
// tool_reference objects or the same plain-text empty message as TS (including MCP connecting suffix
// when [engine.WithPendingMcpServers] / [engine.WithPendingMcpServerNames] populate context).
// [walkToolResultDiscoveryJSON] reads both that array and the legacy {"discovery":[...]} wrapper.
package toolsearch
