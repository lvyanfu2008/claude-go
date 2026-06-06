package config

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"goc/conversation-runtime/query"
	"goc/sessiontranscript"
	"goc/types"
)

// Config is the runtime configuration for the TUI app.
type Config struct {
	// SessionID is the conversation session ID. Auto-generated when empty.
	SessionID string
	// PermissionMode sets the tool permission mode for the session.
	PermissionMode types.PermissionMode
	// CWD is the working directory. Defaults to os.Getwd() when empty.
	CWD string
	// TranscriptPath is an optional JSON file to load initial messages from.
	TranscriptPath string
	// ReplayCCPath is an optional NDJSON stream file to replay before starting the TUI.
	ReplayCCPath string
	// StreamStdin feeds NDJSON stream events from stdin before opening the TUI.
	StreamStdin bool
	// MCPCommandsJSONPath overrides the path for MCP command definitions.
	MCPCommandsJSONPath string
	// MCPToolsJSONPath overrides the path for MCP tool definitions.
	MCPToolsJSONPath string
}

// Validate sets defaults for zero-valued fields.
func (c *Config) Validate() error {
	if c.CWD == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("config: getcwd: %w", err)
		}
		c.CWD = cwd
	}
	if c.SessionID == "" || !sessiontranscript.IsValidUUID(c.SessionID) {
		c.SessionID = sessiontranscript.NewUUID()
	}
	return nil
}

// RunQueryStreamingParityTurn runs [query.Query] in a goroutine and forwards whole messages to the Bubble Tea program.
func RunQueryStreamingParityTurn(ctx context.Context, programSend func(tea.Msg), qp query.QueryParams) {
	go func() {
		for y, err := range query.Query(ctx, qp) {
			if err != nil {
				if programSend != nil {
					programSend(QueryDoneMsg{Err: err})
				}
				return
			}
			if y.StreamEvent != nil && programSend != nil {
				programSend(StreamEventMsg{Raw: y.StreamEvent})
			}
			if y.Message != nil && programSend != nil {
				programSend(QueryYieldMsg{Message: *y.Message})
			}
			if y.Terminal != nil {
				// Query encodes model/stream failures on Terminal.Error (second iter return is always nil err).
				var doneErr error
				if y.Terminal.Error != nil {
					doneErr = y.Terminal.Error
				}
				if programSend != nil {
					programSend(QueryDoneMsg{Err: doneErr})
				}
				return
			}
		}
	}()
}
