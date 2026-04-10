// Command gou-api-parity-snapshot prints JSON: tools[], system, user_context_reminder (prepend-only), and sha256 digests
// for the gou-demo / localturn API slice (system via [querycontext.FetchSystemPromptParts]).
// Use the same env as gou-demo; also reads project .claude/settings*.json for language/outputStyle when -cwd is set (user home skipped for stable hashes). CLAUDE_CODE_* overrides. Built-in outputStyle keys: Explanatory, Learning.
// Other env: CLAUDE_CODE_LANGUAGE, CLAUDE_CODE_OUTPUT_STYLE_*, CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME,
// GOU_DEMO_NON_INTERACTIVE, FEATURE_MCP_SKILLS — compare against TS captures (see docs/plans/go-ts-phase-3-and-gou-demo-runtime.md § 验收).
//
// Run: cd goc && go run ./cmd/gou-api-parity-snapshot [flags]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"goc/apiparity"
	"goc/commands"
	"goc/types"
)

func main() {
	deterministic := flag.Bool("deterministic-env", false, "use linux/amd64 in # Environment information for stable hashes")
	cwdFlag := flag.String("cwd", "", "primary working directory (default: os.Getwd)")
	modelFlag := flag.String("model", "", "main loop model id (default: claude-sonnet-4-20250514)")
	loadCommands := flag.Bool("load-commands", false, "merge disk skills via commands.LoadAndFilterCommands (non-deterministic across repos)")
	mcpPath := flag.String("mcp-commands-json", "", "optional JSON array of types.Command for MCP merge (tests / bridge)")
	compact := flag.Bool("compact", false, "single-line JSON")
	flag.Parse()

	in := apiparity.SnapshotInput{
		Cwd:           *cwdFlag,
		MainLoopModel: *modelFlag,
	}
	if *deterministic {
		in.ParityGOOS = "linux"
		in.ParityGOARCH = "amd64"
	}

	if *mcpPath != "" {
		data, err := os.ReadFile(*mcpPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read mcp json: %v\n", err)
			os.Exit(1)
		}
		var mcp []types.Command
		if err := json.Unmarshal(data, &mcp); err != nil {
			fmt.Fprintf(os.Stderr, "parse mcp json: %v\n", err)
			os.Exit(1)
		}
		in.MCPCommands = mcp
	}

	if *loadCommands {
		cwd := in.Cwd
		if cwd == "" {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
				os.Exit(1)
			}
			in.Cwd = cwd
		}
		loaded, err := commands.LoadAndFilterCommands(context.Background(), cwd, commands.DefaultLoadOptions(), commands.DefaultConsoleAPIAuth())
		if err != nil {
			fmt.Fprintf(os.Stderr, "load commands: %v\n", err)
			os.Exit(1)
		}
		in.LoadedCommands = loaded
	}

	out, err := apiparity.GouDemo(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot: %v\n", err)
		os.Exit(1)
	}

	var enc []byte
	if *compact {
		enc, err = json.Marshal(out)
	} else {
		enc, err = json.MarshalIndent(out, "", "  ")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(enc); err != nil {
		os.Exit(1)
	}
	if !*compact {
		_, _ = os.Stdout.WriteString("\n")
	}
}
