// Package llm abstracts Anthropic Messages vs OpenAI-compatible (e.g. DeepSeek) chat completions.
package llm

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
