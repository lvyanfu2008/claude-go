// Package llmturn provides TurnCompleter implementations (Anthropic Messages vs OpenAI chat/completions)
// for ccb-engine Session.RunTurn and socketserve — not the primary gou-demo path, which uses
// goc/conversation-runtime/query.
package llmturn

import (
	"context"

	"goc/ccb-engine/internal/anthropic"
)

// TurnResult is one model reply in internal (Anthropic-shaped) blocks.
type TurnResult struct {
	Blocks       []anthropic.ContentBlock
	StopReason   string // "end_turn" | "tool_use" (normalized)
	InputTokens  int
	OutputTokens int
}

// TurnCompleter performs one completion step against the current transcript.
type TurnCompleter interface {
	Complete(ctx context.Context, messages []anthropic.Message, tools []anthropic.ToolDefinition, system string) (*TurnResult, error)
}
