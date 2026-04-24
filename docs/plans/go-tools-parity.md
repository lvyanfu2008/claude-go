# gou-demo local tools parity (Go vs TS)

The gou-demo TUI streams tool calls through [`goc/tools/skilltools.ParityToolRunner`](../../tools/skilltools/parity_runner.go) (REPL batch path: `parity_runner_repl.go`). Tool envelopes and behavior are tracked in [`tools/toolparity/TS_GO_TOOL_PARITY.md`](../../tools/toolparity/TS_GO_TOOL_PARITY.md) (regenerate with `go run ./cmd/gen-tool-parity`).

## Environment gates (subset)

| Concern | Default | Override |
|--------|---------|----------|
| **Bash** | Allowed (TS-aligned; gou-demo uses `LocalBashDefault`) | `GOU_DEMO_NO_LOCAL_BASH=1` or `CCB_ENGINE_DISABLE_LOCAL_BASH=1` disables; hosts without local default need `CCB_ENGINE_LOCAL_BASH=1` |
| **PowerShell** | Off | `CCB_ENGINE_LOCAL_POWERSHELL=1` enables (`pwsh` or `powershell.exe`) |
| **AskUserQuestion** | Auto-picks first option per question | `GOU_DEMO_NO_ASK_AUTO_FIRST=1` disables auto-pick |
| **WebFetch** | Allowed | `CCB_ENGINE_DISABLE_WEB_FETCH=1` blocks network fetches in the Go runner |

See also the block comment at the top of [`cmd/gou-demo/main.go`](../../cmd/gou-demo/main.go).
