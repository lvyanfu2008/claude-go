// Command ccb-socket-host runs only the Unix-socket SubmitUserTurn protocol (goc/ccb-engine/socketserve).
// For interactive use, prefer gou-demo (embeds the same listener). This binary is for headless automation (e.g. scripts/ccb-worker-daemon.sh).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/settingsfile"
	"goc/ccb-engine/socketserve"
)

func main() {
	if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
		fmt.Fprintf(os.Stderr, "ccb-socket-host: %v\n", err)
		os.Exit(1)
	}
	apilog.PrepareIfEnabled()
	apilog.MaybePrintDiag()

	socketPath := flag.String("socket", "", "Unix domain socket path (or set CCB_ENGINE_SOCKET)")
	flag.Parse()
	path := *socketPath
	if path == "" {
		path = os.Getenv("CCB_ENGINE_SOCKET")
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "ccb-socket-host: need -socket or CCB_ENGINE_SOCKET")
		os.Exit(2)
	}
	logf := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	if err := socketserve.Run(context.Background(), path, logf); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "ccb-socket-host: %v\n", err)
		os.Exit(1)
	}
}
