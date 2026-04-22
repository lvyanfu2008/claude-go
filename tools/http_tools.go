package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	maxWebFetchBody   = 10 << 20 // TS MAX_HTTP_CONTENT_LENGTH
	maxWebFetchResult = 100_000  // TS tool truncation / markdown cap
	webFetchTimeout   = 60 * time.Second
	maxRedirects      = 10
)

func webFetchDisabled() bool {
	return envTruthy("CCB_ENGINE_DISABLE_WEB_FETCH")
}

func stripWWWHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimPrefix(host, "www.")
}

// webFetchPermittedRedirect mirrors isPermittedRedirect (WebFetchTool/utils.ts).
func webFetchPermittedRedirect(originalURL, redirectURL string) bool {
	o, err1 := url.Parse(originalURL)
	r, err2 := url.Parse(redirectURL)
	if err1 != nil || err2 != nil {
		return false
	}
	if r.Scheme != o.Scheme {
		return false
	}
	if r.Port() != o.Port() {
		return false
	}
	if r.User != nil && r.User.String() != "" {
		return false
	}
	return stripWWWHost(o.Hostname()) == stripWWWHost(r.Hostname())
}

// WebFetchFromJSON performs HTTP GET with same-host redirect policy and TS-shaped {data: ...} output.
// HTML→markdown and prompt-driven extraction use the TS worker + model; this path returns raw text.
func WebFetchFromJSON(ctx context.Context, raw []byte) (string, bool, error) {
	if webFetchDisabled() {
		return "", true, fmt.Errorf("WebFetch disabled (set CCB_ENGINE_DISABLE_WEB_FETCH=0 or unset to allow)")
	}
	var in struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	orig := strings.TrimSpace(in.URL)
	if orig == "" {
		return "", true, fmt.Errorf("url is required")
	}
	start := time.Now()

	client := &http.Client{
		Timeout: webFetchTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	current := orig
	var lastReq *http.Request
	for n := 0; n <= maxRedirects; n++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return "", true, err
		}
		req.Header.Set("User-Agent", "ccb-engine-gou-demo/1.0")
		lastReq = req
		resp, err := client.Do(req)
		if err != nil {
			return "", true, err
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := strings.TrimSpace(resp.Header.Get("Location"))
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if loc == "" {
				return "", true, fmt.Errorf("redirect status %d without Location header", resp.StatusCode)
			}
			base := lastReq.URL
			nextU, err := base.Parse(loc)
			if err != nil {
				return "", true, err
			}
			nextAbs := nextU.String()
			if !webFetchPermittedRedirect(orig, nextAbs) {
				msg := fmt.Sprintf(
					`REDIRECT DETECTED: The URL redirects to a different host.

Original URL: %s
Redirect URL: %s
Status: %d %s

To complete your request, I need to fetch content from the redirected URL. Please use WebFetch again with these parameters:
- url: %q
- prompt: %q`,
					orig, nextAbs, resp.StatusCode, http.StatusText(resp.StatusCode), nextAbs, strings.TrimSpace(in.Prompt),
				)
				out := map[string]any{
					"data": map[string]any{
						"bytes":      len(msg),
						"code":       resp.StatusCode,
						"codeText":   http.StatusText(resp.StatusCode),
						"result":     msg,
						"durationMs": time.Since(start).Milliseconds(),
						"url":        orig,
					},
				}
				b, _ := json.Marshal(out)
				return string(b), false, nil
			}
			current = nextAbs
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebFetchBody+1))
		_ = resp.Body.Close()
		if err != nil {
			return "", true, err
		}
		if len(body) > maxWebFetchBody {
			return "", true, fmt.Errorf("response body exceeds %d bytes", maxWebFetchBody)
		}
		trunc := string(body)
		if len(trunc) > maxWebFetchResult {
			trunc = trunc[:maxWebFetchResult] + "\n…[truncated]"
		}
		result := trunc
		if p := strings.TrimSpace(in.Prompt); p != "" {
			result += "\n\n---\nThe model-side `prompt` extraction from WebFetchTool is not run in the Go runner; pass this page text and the prompt to the main model instead.\nPrompt: " + p
		}
		out := map[string]any{
			"data": map[string]any{
				"bytes":      len(body),
				"code":       resp.StatusCode,
				"codeText":   http.StatusText(resp.StatusCode),
				"result":     result,
				"durationMs": time.Since(start).Milliseconds(),
				"url":        orig,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}
	return "", true, fmt.Errorf("too many redirects (max %d)", maxRedirects)
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

// WebSearchFromJSON mirrors WebSearchTool output shape ({data: ...}); uses ANTHROPIC_WEB_SEARCH_URL when configured.
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
	if q == "" {
		return "", true, fmt.Errorf("Error: Missing query")
	}
	if len(q) < 2 {
		return "", true, fmt.Errorf("query must be at least 2 characters")
	}
	if len(in.AllowedDomains) > 0 && len(in.BlockedDomains) > 0 {
		return "", true, fmt.Errorf("Error: Cannot specify both allowed_domains and blocked_domains in the same request")
	}

	start := time.Now()
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_WEB_SEARCH_URL"))
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if base == "" || key == "" {
		out := map[string]any{
			"data": map[string]any{
				"query":           q,
				"results":         []any{"No search results found."},
				"durationSeconds": time.Since(start).Seconds(),
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	bodyBytes, err := json.Marshal(map[string]string{"query": q})
	if err != nil {
		return "", true, err
	}
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

	var results []any
	var hits []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}
	if err := json.Unmarshal(body, &hits); err == nil && len(hits) > 0 {
		content := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			m := map[string]any{"title": h.Title, "url": h.URL}
			if strings.TrimSpace(h.Snippet) != "" {
				m["snippet"] = h.Snippet
			}
			content = append(content, m)
		}
		results = []any{map[string]any{
			"tool_use_id": "adapter-search-1",
			"content":     content,
		}}
	} else {
		var nested []map[string]any
		if err := json.Unmarshal(body, &nested); err == nil && len(nested) > 0 {
			results = []any{map[string]any{
				"tool_use_id": "adapter-search-1",
				"content":     nested,
			}}
		} else {
			results = []any{string(body)}
		}
	}

	// Domain filters (best-effort on structured hits only).
	if len(in.AllowedDomains) > 0 || len(in.BlockedDomains) > 0 {
		allowed := map[string]struct{}{}
		for _, d := range in.AllowedDomains {
			allowed[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
		}
		blocked := map[string]struct{}{}
		for _, d := range in.BlockedDomains {
			blocked[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
		}
		filtered := filterSearchResults(results, allowed, blocked)
		results = filtered
	}

	out := map[string]any{
		"data": map[string]any{
			"query":           q,
			"results":         results,
			"durationSeconds": time.Since(start).Seconds(),
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func filterSearchResults(results []any, allowed, blocked map[string]struct{}) []any {
	out := make([]any, 0, len(results))
	for _, r := range results {
		m, ok := r.(map[string]any)
		if !ok {
			out = append(out, r)
			continue
		}
		content, _ := m["content"].([]any)
		var nextContent []any
		for _, row := range content {
			rowm, ok := row.(map[string]any)
			if !ok {
				continue
			}
			u, _ := rowm["url"].(string)
			host := hostFromURL(u)
			if len(blocked) > 0 {
				if _, bad := blocked[host]; bad {
					continue
				}
			}
			if len(allowed) > 0 {
				if _, ok := allowed[host]; !ok {
					continue
				}
			}
			nextContent = append(nextContent, rowm)
		}
		if len(nextContent) == 0 {
			out = append(out, "No search results found.")
			continue
		}
		m2 := cloneMapShallow(m)
		m2["content"] = nextContent
		out = append(out, m2)
	}
	if len(out) == 0 {
		return []any{"No search results found."}
	}
	return out
}

func cloneMapShallow(m map[string]any) map[string]any {
	o := make(map[string]any, len(m))
	for k, v := range m {
		o[k] = v
	}
	return o
}

func hostFromURL(s string) string {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil {
		return strings.ToLower(s)
	}
	return strings.ToLower(u.Hostname())
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
