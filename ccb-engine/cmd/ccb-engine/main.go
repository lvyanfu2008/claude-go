// Command ccb-engine is a headless smoke runner for the Go session engine (see README).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/engine"
	"goc/ccb-engine/internal/protocol"
	"goc/ccb-engine/settingsfile"
	"goc/ccb-engine/llmturn"
)

func initClaudeProjectEnv() {
	if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
		fmt.Fprintf(os.Stderr, "ccb-engine: %v\n", err)
		os.Exit(1)
	}
	apilog.PrepareIfEnabled()
	apilog.MaybePrintDiag()
}

func main() {
	initClaudeProjectEnv()

	prompt := flag.String("prompt", "", "user message to append and run one turn")
	stubTools := flag.Bool("stub-tools", false, "register echo_stub tool for model")
	jsonEvents := flag.Bool("json-events", false, "print StreamEvent objects as JSON lines to stderr")
	flag.Parse()

	if *prompt == "" {
		fmt.Fprintln(os.Stderr, "usage: ccb-engine -prompt '...' [-stub-tools] [-json-events]")
		os.Exit(2)
	}

	completer := llmturn.NewFromEnv()

	var sink engine.EventSink
	if *jsonEvents {
		sink = func(ev protocol.StreamEvent) {
			b, err := json.Marshal(ev)
			if err != nil {
				return
			}
			fmt.Fprintln(os.Stderr, string(b))
		}
	}

	sess := engine.NewSession(sink)
	rev := sess.AppendUserText(*prompt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var tools []anthropic.ToolDefinition
	toolsSource := "none"
	if *stubTools {
		tools = anthropic.DefaultStubTools()
		toolsSource = "default_stub"
	}
	anthropic.LogToolsLoaded("ccb-engine-cli", "", toolsSource, tools)

	if err := sess.RunTurn(ctx, completer, tools, "", engine.StubRunner{}, false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "ok user_state_rev=%d final_state_rev=%d\n", rev, sess.StateRev())
}
