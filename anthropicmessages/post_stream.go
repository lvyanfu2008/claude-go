package anthropicmessages

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiVersion = "2023-06-01"

// PostStreamParams configures POST /v1/messages with a JSON body that must include "stream":true.
type PostStreamParams struct {
	BaseURL string
	APIKey  string
	Body    []byte
	HTTP    *http.Client
	Beta    []string // anthropic-beta header, optional
	Emit    func(MessageStreamEvent) error
}

// PostStream performs a streaming Messages request and invokes Emit for each event until message_stop.
func PostStream(ctx context.Context, p PostStreamParams) error {
	if strings.TrimSpace(p.APIKey) == "" {
		return fmt.Errorf("anthropicmessages: missing API key")
	}
	base := strings.TrimSuffix(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("anthropicmessages: missing base URL")
	}
	if p.HTTP == nil {
		p.HTTP = http.DefaultClient
	}
	if p.Emit == nil {
		return fmt.Errorf("anthropicmessages: nil Emit")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(p.Body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")
	if len(p.Beta) > 0 {
		httpReq.Header.Set("anthropic-beta", strings.Join(p.Beta, ","))
	}

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("anthropic stream API %s: %s", resp.Status, truncate(string(b), 800))
	}

	return ReadSSE(resp.Body, func(data []byte) error {
		return ProcessStreamPayloads(data, p.Emit)
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
