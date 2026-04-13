package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
)

// OpenAIPostStreamParams configures POST /v1/chat/completions (streaming).
type OpenAIPostStreamParams struct {
	BaseURL string
	APIKey  string
	Body    []byte
	HTTP    *http.Client
	Emit    func(anthropicmessages.MessageStreamEvent) error
}

// PostOpenAIChatStream POSTs JSON with stream:true and feeds each SSE data object through
// [openAIStreamAdapter.HandleChunk], then [openAIStreamAdapter.FlushOpenBlocks] (TS streamAdapter.ts tail).
func PostOpenAIChatStream(ctx context.Context, p OpenAIPostStreamParams) error {
	if strings.TrimSpace(p.APIKey) == "" {
		return fmt.Errorf("query openai: missing OPENAI_API_KEY")
	}
	base := strings.TrimSpace(p.BaseURL)
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if p.HTTP == nil {
		p.HTTP = http.DefaultClient
	}
	if p.Emit == nil {
		return fmt.Errorf("query openai: nil Emit")
	}
	if apilog.ApiBodyLoggingEnabled() {
		apilog.PrepareIfEnabled()
	}
	url := strings.TrimSuffix(base, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(p.Body))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(p.APIKey))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "text/event-stream")

	apilog.LogRequestBody("POST "+url+" (stream)", p.Body)

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		apilog.LogResponseBody("POST "+url+" (stream error "+resp.Status+")", b)
		return fmt.Errorf("openai chat stream %s: %s", resp.Status, truncateOpenAIErr(string(b), 800))
	}

	streamRd := io.ReadCloser(resp.Body)
	var sseCapture *bytes.Buffer
	if apilog.ResponseBodyLoggingEnabled() {
		sseCapture = &bytes.Buffer{}
		streamRd = anthropicmessages.NewStreamBodyReadTee(resp.Body, sseCapture, anthropicmessages.MaxStreamBodyLogBytes)
	}
	defer func() { _ = streamRd.Close() }()

	var reqHead struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(p.Body, &reqHead)
	ad := newOpenAIStreamAdapter(reqHead.Model)

	if err := anthropicmessages.ReadSSE(streamRd, func(data []byte) error {
		if len(data) == 0 || string(data) == "[DONE]" {
			return nil
		}
		return ad.HandleChunk(data, p.Emit)
	}); err != nil {
		return err
	}
	if sseCapture != nil && sseCapture.Len() > 0 {
		label := "POST " + url + " (SSE stream)"
		if sseCapture.Len() >= anthropicmessages.MaxStreamBodyLogBytes {
			label += fmt.Sprintf(" [truncated after %d bytes]", anthropicmessages.MaxStreamBodyLogBytes)
		}
		apilog.LogResponseBody(label, sseCapture.Bytes())
	}
	return ad.FlushOpenBlocks(p.Emit)
}

func truncateOpenAIErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
