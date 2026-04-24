# REPL tool: Go permissions vs TS (`permissions.ts`)

## TS behavior (reference)

In [`claude-code/src/utils/permissions/permissions.ts`](../../../claude-code/src/utils/permissions/permissions.ts) (around lines 593–604), **`REPL`** and **`Agent`** are excluded from a shortcut that would skip the **auto-mode classifier** when the same tool call would already be **`allow`** under synthetic **`acceptEdits`** permission context. The comment states that REPL “glue” can escape the VM between inner tool calls, so the classifier must see the full REPL payload, not only isolated inner primitives.

Separately, **rule-based** checks (`checkRuleBasedPermissions` / whole-tool alwaysAsk / Bash sandbox **1b**) apply per **tool name + input** at invocation time.

## Go behavior (`goc/tools/toolexecution` + `skilltools`)

1. **Outer `REPL` `tool_use`**  
   When the model calls `REPL`, [`RunToolUseChan`](../../tools/toolexecution/run_tool_use.go) runs the usual pipeline: optional `QueryCanUseTool` → [`applyRuleBasedDecisionInRun`](../../tools/toolexecution/run_tool_use.go) (deny/ask from merged rules + Bash **1b** bypass on the **REPL** name/input) → **`ExecutionDeps.InvokeTool`** → [`ParityToolRunner.Run`](../../tools/skilltools/parity_runner.go) → [`runREPLTool`](../../tools/skilltools/parity_runner_repl.go).

2. **Inner primitives (Read, Bash, …)**  
   REPL execution dispatches inner tools via [`dispatchTool`](../../tools/skilltools/parity_runner.go) **without** re-entering `RunToolUseChan`. Those calls **do not** receive a second `QueryCanUseTool` / `applyRuleBasedDecisionInRun` pass at the toolexecution layer. Inner Bash still uses [`localtools.BashFromJSON`](../../tools/localtools/bash.go) and its own permission / sandbox behavior where implemented.

3. **No TS auto-mode classifier in Go**  
   The Go stack does not implement the TS “auto mode classifier” or the **acceptEdits short-circuit** branch. There is therefore **nothing to exclude** `REPL` from in that layer; parity is “N/A / structural difference”, not a silent bypass of a missing step.

## Known deltas (intentional for this milestone)

| Area | TS | Go |
|------|----|-----|
| Auto-mode classifier + acceptEdits shortcut | Present; REPL/Agent excluded | Not implemented |
| Permission passes per inner REPL primitive | Enforced in TS VM as configured | Single outer gate + `dispatchTool` (documented here) |
| ReplBridge / remote REPL | Optional | Out of scope — see [gou-demo-repl-bridge-scope.md](./gou-demo-repl-bridge-scope.md) |

Future work: optionally route each inner `dispatchTool` call through a small adapter that calls `toolexecution.CheckRuleBasedPermissions` (or a trimmed `InvokeTool`) with a synthetic `tool_use_id` suffix for parity with stricter TS deployments.
