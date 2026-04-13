// Command export-bashzog-json writes the BashZog API-shaped tool row for diffing vs TS / tools_api.
package main

import (
	"flag"
	"fmt"
	"os"

	"goc/ccb-engine/bashzog"
)

func main() {
	stdout := flag.Bool("stdout", false, "write JSON to stdout instead of -out")
	outPath := flag.String("out", "ccb-engine/bashzog/bash_zog_tool_export.json", "output path (relative to cwd, typically claude-go module root)")
	flag.Parse()

	b, err := bashzog.ExportBashZogToolJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "export: %v\n", err)
		os.Exit(1)
	}
	if *stdout {
		if _, err := os.Stdout.Write(b); err != nil {
			fmt.Fprintf(os.Stderr, "write stdout: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*outPath, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
}
