// Command slash-prepare reads JSON on stdin { "input" } and writes prepare Result JSON on stdout (Phase 1: parse only).
package main

import (
	"fmt"
	"io"
	"os"

	slashprepare "goc/slash-prepare"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "slash-prepare: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	out, err := slashprepare.Run(data)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	if err != nil {
		return fmt.Errorf("write stdout: %w", err)
	}
	return nil
}
