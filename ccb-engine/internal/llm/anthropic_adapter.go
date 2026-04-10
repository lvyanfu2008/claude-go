package llm

import (
	"context"

	"goc/ccb-engine/internal/anthropic"
)

// AnthropicAdapter wraps the Anthropic Messages API client.
type AnthropicAdapter struct {
	Client *anthropic.Client
}

func (a *AnthropicAdapter) Complete(ctx context.Context, messages []anthropic.Message, tools []anthropic.ToolDefinition, system string) (*TurnResult, error) {
	resp, err := a.Client.CreateMessage(ctx, anthropic.CreateMessageRequest{
		Messages: messages,
		Tools:    tools,
		System:   system,
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
