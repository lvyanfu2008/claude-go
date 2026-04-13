# ReplBridge vs gou-demo (M3 scope)

TS components [`claude-code/src/bridge/replBridge.ts`](../../../claude-code/src/bridge/replBridge.ts), [`replBridgeTransport.ts`](../../../claude-code/src/bridge/replBridgeTransport.ts), and [`useReplBridge.tsx`](../../../claude-code/src/hooks/useReplBridge.tsx) implement **remote** REPL / control-plane messaging (session ingress, SDK control requests, capacity wake, etc.).

## Non-goal for gou-demo / `goc` TUI

**gou-demo does not implement ReplBridge clients.** Local tool execution uses [`goc/ccb-engine/skilltools.ParityToolRunner`](../../ccb-engine/skilltools/parity_runner.go) (including the **REPL** batch runner in `parity_runner_repl.go`). There is no WebSocket/SDK bridge from the Bubble Tea demo to a remote Claude Code host.

## When a Go ReplBridge would matter

Only if a future product path requires the same **remote REPL** semantics as Ink (e.g. `writeMessages` / `sendControlRequest` against session ingress). That would be a **separate** package (HTTP/WebSocket client) and is **out of scope** for the transcript + local REPL primitive milestone described in [`gou-demo-transcript-ts-parity.md`](./gou-demo-transcript-ts-parity.md).
