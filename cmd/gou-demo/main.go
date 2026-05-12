// Command gou-demo is the Bubble Tea full-screen TUI for Claude Code Go.
// It delegates to goc/gou/app for all TUI logic.
//
// Run from repo: cd goc && go run ./cmd/gou-demo
package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"goc/gou/app"
	"goc/sessiontranscript"
	"goc/types"
)

func main() {
	transcriptPath := flag.String("transcript", "", "load initial messages from JSON file")
	replayCC := flag.String("replay-cc", "", "replay ccb-engine NDJSON stream from file")
	streamStdin := flag.Bool("stream-stdin", false, "read NDJSON stream events from stdin")
	mcpCommandsJSON := flag.String("mcp-commands-json", "", "JSON file of MCP prompt commands")
	mcpToolsJSON := flag.String("mcp-tools-json", "", "JSON file of MCP tool definitions")
	flag.Parse()

	cfg := app.Config{
		TranscriptPath:      *transcriptPath,
		ReplayCCPath:        *replayCC,
		StreamStdin:         *streamStdin,
		MCPCommandsJSONPath: *mcpCommandsJSON,
		MCPToolsJSONPath:    *mcpToolsJSON,
		PermissionMode:      types.PermissionMode(strings.TrimSpace(os.Getenv("CLAUDE_CODE_PERMISSION_MODE"))),
	}

	if sid := strings.TrimSpace(os.Getenv("GOU_DEMO_SESSION_ID")); sessiontranscript.IsValidUUID(sid) {
		cfg.SessionID = sid
	}

	if err := app.Run(cfg); err != nil {
		log.Fatalf("gou-demo: %v", err)
	}
}
