package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"goc/ccb-engine/apilog"
	"goc/modelenv"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
)

// Client calls Anthropic Messages API (non-streaming).
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	Model   string
}

func NewClient() *Client {
	base := os.Getenv("ANTHROPIC_BASE_URL")
	if base == "" {
		base = defaultBaseURL
	}
	model := modelenv.ResolveWithFallback("claude-sonnet-4-20250514")
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		key = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	return &Client{
		APIKey:  key,
		BaseURL: strings.TrimSuffix(base, "/"),
		HTTP:    http.DefaultClient,
		Model:   model,
	}
}

// ToolDefinition is the tools[] entry for the Messages API.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

// CreateMessageRequest is POST /v1/messages body.
type CreateMessageRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []Message        `json:"messages"`
	System    string           `json:"system,omitempty"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
	Stream    bool             `json:"stream,omitempty"`
}

// CreateMessageResponse is a subset of the API response.
type CreateMessageResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// CreateMessage performs a non-streaming messages request.
func (c *Client) CreateMessage(ctx context.Context, req CreateMessageRequest) (*CreateMessageResponse, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("set ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN (shell, or project .claude/settings.json env when EnsureProjectClaudeEnvOnce has run)")
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	req.Stream = false

	body, err := marshalJSONNoEscapeHTML(req)
	if err != nil {
		return nil, err
	}
	apilog.LogRequestBody("POST "+c.BaseURL+"/v1/messages", body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apilog.LogResponseBody("POST "+c.BaseURL+"/v1/messages", respBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic API %s: %s", resp.Status, truncate(string(respBody), 800))
	}

	var out CreateMessageResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// marshalJSONNoEscapeHTML matches JSON.stringify / TS request logging: keep literal < in strings.
func marshalJSONNoEscapeHTML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// ParseContentBlocks decodes assistant content JSON array into blocks.
func ParseContentBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// DefaultStubTools returns tools used by the engine for stub tool-use loops.
func DefaultStubTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "echo_stub",
			Description: "Echo a short message; the engine answers with a stub tool_result for wiring tests.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{"type": "string", "description": "Text to echo"},
				},
				"required": []string{"message"},
			},
		},
	}
}
