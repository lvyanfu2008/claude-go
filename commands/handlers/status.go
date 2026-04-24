package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
)

// StatusResult is the JSON payload returned by /status.
type StatusResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleStatusCommand returns Claude Code status info for /status.
// Mirrors TS src/commands/status/ (local-jsx -> Settings > Status tab).
func HandleStatusCommand() ([]byte, error) {
	version := "gou-demo (dev)"
	if bi, ok := debug.ReadBuildInfo(); ok {
		v := bi.Main.Version
		if v != "" && v != "(devel)" {
			version = v
		}
	}

	model := os.Getenv("CLAUDE_CODE_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6 (default)"
	}

	effort := os.Getenv("CLAUDE_CODE_EFFORT_LEVEL")
	effortStr := "auto"
	if effort != "" {
		effortStr = effort
	}

	lines := []string{
		"Claude Code (gou-demo) — Status",
		"═════════════════════════════════",
		fmt.Sprintf("  Version:   %s", version),
		fmt.Sprintf("  OS:        %s / %s", runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("  Go:        %s", runtime.Version()),
		fmt.Sprintf("  Model:     %s", model),
		fmt.Sprintf("  Effort:    %s", effortStr),
	}

	// Check API connectivity by proxy (env vars)
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		lines = append(lines, "  API:       ✓ ANTHROPIC_API_KEY set")
	} else {
		lines = append(lines, "  API:       ✗ ANTHROPIC_API_KEY not set")
	}

	lines = append(lines, "")
	lines = append(lines, "For detailed status, use the TS CLI: claude /status")

	msg := StatusResult{
		Type:  "text",
		Value: joinLines(lines),
	}
	return json.Marshal(msg)
}
