// Command export-bashzog-json writes the BashZog API-shaped tool row (e.g. for ad-hoc diff vs TS).
package main

import (
	"flag"
	"fmt"
	"os"

	"goc/ccb-engine/bashzog"
)

func main() {
	outPath := flag.String("out", "-", `output path, or "-" for stdout (default)`)
	flag.Parse()

	b, err := bashzog.ExportBashZogToolJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "export: %v\n", err)
		os.Exit(1)
	}
	if *outPath == "-" {
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
