package llm

import (
	"context"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/toolsearch"
)

// AnthropicAdapter wraps the Anthropic Messages API client.
type AnthropicAdapter struct {
	Client *anthropic.Client
}

func (a *AnthropicAdapter) Complete(ctx context.Context, messages []anthropic.Message, tools []anthropic.ToolDefinition, system string) (*TurnResult, error) {
	msgs := anthropic.CanonicalizeMessages(messages)
	resp, err := a.Client.CreateMessage(ctx, anthropic.CreateMessageRequest{
		Messages:      msgs,
		Tools:         tools,
		System:        system,
		AnthropicBeta: toolsearch.BetasForWiredTools(tools),
	})
	if err != nil {
		return nil, err
	}
	blocks, err := anthropic.ParseContentBlocks(resp.Content)
	if err != nil {
		return nil, err
	}
	return &TurnResult{
		Blocks:       blocks,
		StopReason:   resp.StopReason,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}
