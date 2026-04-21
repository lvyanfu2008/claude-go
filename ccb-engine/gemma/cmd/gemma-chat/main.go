// Command gemma-chat is a minimal terminal chat that calls the Vertex Gemma HTTP client
// in goc/ccb-engine/gemma. It is optional dev tooling and not used by the main Claude Go runtime.
package main

import (
	"fmt"
	"os"

	"goc/ccb-engine/gemma/chatui"
)

func main() {
	if err := chatui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
