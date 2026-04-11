package anthropicmessages

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"goc/ccb-engine/apilog"
)

const apiVersion = "2023-06-01"

// maxStreamResponseLogBytes caps raw SSE bytes logged for one stream when
// CLAUDE_CODE_LOG_API_RESPONSE_BODY is on (avoids unbounded memory).
const maxStreamResponseLogBytes = 32 << 20

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

	if apilog.ApiBodyLoggingEnabled() {
		apilog.PrepareIfEnabled()
	}

	url := base + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(p.Body))
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

	apilog.LogRequestBody("POST "+url+" (stream)", p.Body)

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		apilog.LogResponseBody("POST "+url+" (stream error "+resp.Status+")", b)
		return fmt.Errorf("anthropic stream API %s: %s", resp.Status, truncate(string(b), 800))
	}

	streamRd := io.ReadCloser(resp.Body)
	var sseCapture *bytes.Buffer
	if apilog.ResponseBodyLoggingEnabled() {
		sseCapture = &bytes.Buffer{}
		streamRd = &sseStreamCapture{
			rc:  resp.Body,
			buf: sseCapture,
			max: maxStreamResponseLogBytes,
		}
	}
	defer func() { _ = streamRd.Close() }()

	if err := ReadSSE(streamRd, func(data []byte) error {
		return ProcessStreamPayloads(data, p.Emit)
	}); err != nil {
		return err
	}
	if sseCapture != nil && sseCapture.Len() > 0 {
		label := "POST " + url + " (SSE stream)"
		if sseCapture.Len() >= maxStreamResponseLogBytes {
			label += fmt.Sprintf(" [truncated after %d bytes]", maxStreamResponseLogBytes)
		}
		apilog.LogResponseBody(label, sseCapture.Bytes())
	}
	return nil
}

// sseStreamCapture tees the raw SSE bytes into buf up to max bytes (for apilog).
type sseStreamCapture struct {
	rc  io.ReadCloser
	buf *bytes.Buffer
	max int
}

func (s *sseStreamCapture) Read(p []byte) (int, error) {
	n, err := s.rc.Read(p)
	if n > 0 && s.buf != nil && s.buf.Len() < s.max {
		room := s.max - s.buf.Len()
		if room > 0 {
			if n <= room {
				_, _ = s.buf.Write(p[:n])
			} else {
				_, _ = s.buf.Write(p[:room])
			}
		}
	}
	return n, err
}

func (s *sseStreamCapture) Close() error {
	return s.rc.Close()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
