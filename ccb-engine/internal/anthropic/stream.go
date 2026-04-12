package anthropic

import (
	"context"
	"fmt"

	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
)

// MessageStreamEvent aliases the public SSE payload type.
type MessageStreamEvent = anthropicmessages.MessageStreamEvent

// ErrMessageStreamDone indicates a normal end after message_stop.
var ErrMessageStreamDone = anthropicmessages.ErrMessageStreamDone

// CreateMessageStream performs POST /v1/messages with "stream":true and invokes emit
// for each SSE JSON payload until message_stop or error.
func (c *Client) CreateMessageStream(ctx context.Context, req CreateMessageRequest, emit func(MessageStreamEvent) error) error {
	if c.APIKey == "" {
		return fmt.Errorf("set ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN")
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	req.Stream = true

	body, err := marshalJSONNoEscapeHTML(req)
	if err != nil {
		return err
	}
	apilog.LogRequestBody("POST "+anthropicmessages.MessagesAPIURL(c.BaseURL)+" (stream)", body)

	return anthropicmessages.PostStream(ctx, anthropicmessages.PostStreamParams{
		BaseURL: c.BaseURL,
		APIKey:  c.APIKey,
		Body:    body,
		HTTP:    c.HTTP,
		Beta:    req.AnthropicBeta,
		Emit:    emit,
	})
}
