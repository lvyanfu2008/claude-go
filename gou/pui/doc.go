// Package pui wires the Go TUI (gou-demo) to process-user-input in-process.
//
// TS naming / contracts (keep in sync when changing types here):
//   - ProcessUserInputBaseResult — src/conversation-runtime/processUserInput/processUserInput.ts
//   - processUserInput(), ProcessUserInputParams — same file + processUserInput.ts exports
//   - ToolUseContext / ProcessUserInputContext (serializable slice) — src/Tool.ts, types/tool_context.go (Go)
//   - goc/cmd/process-user-input stdout shape — goc/cmd/process-user-input/main.go
//
// See goc/gou/README.md §「ProcessUserInput：已做 / 未做」for coverage vs the full REPL path.
package pui
