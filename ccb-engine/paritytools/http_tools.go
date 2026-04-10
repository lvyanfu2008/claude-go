package paritytools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxWebFetchBody = 2 << 20 // 2MB cap (subset vs TS markdown path)

// WebFetchFromJSON performs HTTP GET; does not run an LLM on content (documented parity gap).
func WebFetchFromJSON(ctx context.Context, raw []byte) (string, bool, error) {
	if !envTruthy("CCB_ENGINE_WEB_FETCH") && !envTruthy("GOU_DEMO_WEB_FETCH") {
		return "", true, fmt.Errorf("WebFetch disabled in Go runner (set CCB_ENGINE_WEB_FETCH=1 or GOU_DEMO_WEB_FETCH=1)")
	}
	var in struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	u := strings.TrimSpace(in.URL)
	if u == "" {
		return "", true, fmt.Errorf("url is required")
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", true, err
	}
	req.Header.Set("User-Agent", "ccb-engine-gou-demo/1.0")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", true, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebFetchBody+1))
	if err != nil {
		return "", true, err
	}
	if len(body) > maxWebFetchBody {
		return "", true, fmt.Errorf("response body exceeds %d bytes", maxWebFetchBody)
	}
	trunc := string(body)
	if len(trunc) > 12000 {
		trunc = trunc[:12000] + "\n…[truncated]"
	}
	note := "Go runner returns raw body excerpt; TS applies `prompt` via model — not replicated here."
	if strings.TrimSpace(in.Prompt) != "" {
		note += fmt.Sprintf(" (prompt was: %q)", in.Prompt)
	}
	out := map[string]any{
		"bytes":      len(body),
		"code":       resp.StatusCode,
		"codeText":   http.StatusText(resp.StatusCode),
		"result":     trunc + "\n\n" + note,
		"durationMs": time.Since(start).Milliseconds(),
		"url":        u,
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// WebSearchFromJSON is a stub unless ANTHROPIC_WEB_SEARCH_URL + key are set (experimental).
func WebSearchFromJSON(ctx context.Context, raw []byte) (string, bool, error) {
	var in struct {
		Query          string   `json:"query"`
		AllowedDomains []string `json:"allowed_domains"`
		BlockedDomains []string `json:"blocked_domains"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	q := strings.TrimSpace(in.Query)
	if len(q) < 2 {
		return "", true, fmt.Errorf("query must be at least 2 characters")
	}
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_WEB_SEARCH_URL"))
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if base == "" || key == "" {
		return "", true, fmt.Errorf("WebSearch not configured in Go runner (set ANTHROPIC_WEB_SEARCH_URL + ANTHROPIC_API_KEY, or use TS socket worker)")
	}
	_ = in.AllowedDomains
	_ = in.BlockedDomains
	bodyBytes, err := json.Marshal(map[string]string{"query": q})
	if err != nil {
		return "", true, err
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", true, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("content-type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", true, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return "", true, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", true, fmt.Errorf("web search HTTP %d: %s", resp.StatusCode, truncateStr(string(body), 400))
	}
	out := map[string]any{
		"query":           q,
		"results":         []any{string(body)},
		"durationSeconds": time.Since(start).Seconds(),
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
