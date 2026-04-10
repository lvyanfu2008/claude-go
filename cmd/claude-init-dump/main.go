// Command claude-init-dump runs [claudeinit.Init] and prints [claudeinit.DumpState] JSON (stdout).
// Used with [scripts/dump-init-state.ts] for parity harness.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"goc/claudeinit"
)

func main() {
	cwd := flag.String("cwd", "", "if set, chdir before init")
	flag.Parse()
	opts := claudeinit.Options{
		NonInteractive: true,
		WorkingDir:     *cwd,
	}
	if err := claudeinit.Init(context.Background(), opts); err != nil {
		log.Fatalf("claudeinit: %v", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(claudeinit.DumpState()); err != nil {
		log.Fatal(err)
	}
}
