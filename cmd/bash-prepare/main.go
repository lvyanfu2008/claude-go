// Command bash-prepare reads JSON on stdin { "input", "shell"? } and writes prepare Result JSON on stdout (Phase 1: no shell exec).
package main

import (
	"fmt"
	"io"
	"os"

	bashprepare "goc/bash-prepare"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bash-prepare: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	out, err := bashprepare.Run(data)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	if err != nil {
		return fmt.Errorf("write stdout: %w", err)
	}
	return nil
}
