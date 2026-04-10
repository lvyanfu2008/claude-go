package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/apilog"
)

// OpenAICompat calls POST {base}/chat/completions (Bearer auth). Works with DeepSeek and other OpenAI-style APIs.
type OpenAICompat struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	Model   string
}

func newOpenAICompatFromEnv() *OpenAICompat {
	base := os.Getenv("ANTHROPIC_BASE_URL")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := os.Getenv("CCB_ENGINE_MODEL")
	if model == "" {
		model = os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")
	}
	if model == "" {
		model = "deepseek-chat"
	}
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		key = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAICompat{
		APIKey:  key,
		BaseURL: strings.TrimSuffix(base, "/"),
		HTTP:    http.DefaultClient,
		Model:   model,
	}
}

type oaChatMessage struct {
	Role       string       `json:"role"`
	Content    any          `json:"content,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	Name       string       `json:"name,omitempty"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Parameters  any    `json:"parameters,omitempty"`
	} `json:"function"`
}

type oaChatRequest struct {
	Model     string          `json:"model"`
	Messages  []oaChatMessage `json:"messages"`
	Tools     []oaTool        `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type oaChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role      string          `json:"role"`
			Content   json.RawMessage `json:"content"`
			ToolCalls []oaToolCall    `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func toolsToOpenAI(tools []anthropic.ToolDefinition) []oaTool {
	var out []oaTool
	for _, t := range tools {
		out = append(out, oaTool{
			Type: "function",
			Function: struct {
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
				Parameters  any    `json:"parameters,omitempty"`
			}{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return out
}

func messagesToOpenAI(msgs []anthropic.Message) ([]oaChatMessage, error) {
	var out []oaChatMessage
	for _, m := range msgs {
		switch m.Role {
		case "user":
			switch c := m.Content.(type) {
			case string:
				out = append(out, oaChatMessage{Role: "user", Content: c})
			case []anthropic.ContentBlock:
				out = append(out, userBlocksToOA(c)...)
			default:
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, err
				}
				var blocks []anthropic.ContentBlock
				if err := json.Unmarshal(raw, &blocks); err != nil {
					return nil, fmt.Errorf("user message content: %w", err)
				}
				out = append(out, userBlocksToOA(blocks)...)
			}
		case "assistant":
			switch c := m.Content.(type) {
			case string:
				out = append(out, oaChatMessage{Role: "assistant", Content: c})
			case []anthropic.ContentBlock:
				out = append(out, assistantBlocksToOA(c)...)
			default:
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, err
				}
				var blocks []anthropic.ContentBlock
				if err := json.Unmarshal(raw, &blocks); err != nil {
					return nil, fmt.Errorf("assistant message content: %w", err)
				}
				out = append(out, assistantBlocksToOA(blocks)...)
			}
		default:
			return nil, fmt.Errorf("unsupported message role %q", m.Role)
		}
	}
	return out, nil
}

func toolResultContentString(c any) string {
	switch v := c.(type) {
	case string:
		return v
	default:
		bb, _ := json.Marshal(v)
		return string(bb)
	}
}

// userBlocksToOA maps Anthropic user content blocks to OpenAI user + tool messages (order preserved).
func userBlocksToOA(blocks []anthropic.ContentBlock) []oaChatMessage {
	var out []oaChatMessage
	var pendingText []string
	flushText := func() {
		if len(pendingText) == 0 {
			return
		}
		out = append(out, oaChatMessage{Role: "user", Content: strings.Join(pendingText, "\n")})
		pendingText = nil
	}
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				pendingText = append(pendingText, b.Text)
			}
		case "tool_result":
			flushText()
			out = append(out, oaChatMessage{
				Role:       "tool",
				ToolCallID: b.ToolUseID,
				Content:    toolResultContentString(b.Content),
			})
		default:
			if b.Text != "" {
				pendingText = append(pendingText, b.Text)
			} else if b.Type != "" {
				pendingText = append(pendingText, fallbackBlockSummary(b))
			}
		}
	}
	flushText()
	return out
}

// fallbackBlockSummary turns Claude-only blocks (server_tool_use, advisor_tool_result, …) into plain text
// so OpenAI chat/completions never receives an assistant message with neither content nor tool_calls.
func fallbackBlockSummary(b anthropic.ContentBlock) string {
	line := "[" + b.Type
	if b.Name != "" {
		line += " " + b.Name
	}
	if b.ID != "" {
		line += " id=" + b.ID
	}
	line += "]"
	if len(b.Input) > 0 {
		in := string(b.Input)
		const max = 400
		if len(in) > max {
			in = in[:max] + "…"
		}
		line += " " + in
	}
	return line
}

func assistantBlocksToOA(blocks []anthropic.ContentBlock) []oaChatMessage {
	var textParts []string
	var calls []oaToolCall
	for _, b := range blocks {
		switch b.Type {
		case "text":
			textParts = append(textParts, b.Text)
		case "tool_use":
			args := string(b.Input)
			if args == "" {
				args = "{}"
			}
			calls = append(calls, oaToolCall{
				ID:   b.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: b.Name, Arguments: args},
			})
		default:
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			} else if b.Type != "" {
				textParts = append(textParts, fallbackBlockSummary(b))
			}
		}
	}
	msg := oaChatMessage{Role: "assistant"}
	if len(textParts) > 0 {
		msg.Content = strings.Join(textParts, "")
	}
	if len(calls) > 0 {
		msg.ToolCalls = calls
	}
	if msg.Content == nil && len(msg.ToolCalls) == 0 {
		msg.Content = "(empty assistant turn)"
	}
	return []oaChatMessage{msg}
}

func (o *OpenAICompat) Complete(ctx context.Context, messages []anthropic.Message, tools []anthropic.ToolDefinition, system string) (*TurnResult, error) {
	if o.APIKey == "" {
		return nil, fmt.Errorf("set ANTHROPIC_AUTH_TOKEN, ANTHROPIC_API_KEY, or OPENAI_API_KEY for OpenAI-compatible API")
	}
	oaMsgs, err := messagesToOpenAI(messages)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(system) != "" {
		oaMsgs = append([]oaChatMessage{{Role: "system", Content: system}}, oaMsgs...)
	}
	req := oaChatRequest{
		Model:     o.Model,
		Messages:  oaMsgs,
		MaxTokens: 4096,
	}
	if len(tools) > 0 {
		req.Tools = toolsToOpenAI(tools)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := o.BaseURL + "/chat/completions"
	apilog.LogRequestBody("POST "+url, body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("authorization", "Bearer "+o.APIKey)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := o.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apilog.LogResponseBody("POST "+url, respBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("chat/completions %s: %s", resp.Status, truncateStr(string(respBody), 800))
	}
	var parsed oaChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	ch := parsed.Choices[0]
	var blocks []anthropic.ContentBlock
	if len(ch.Message.Content) > 0 && string(ch.Message.Content) != "null" {
		var s string
		if err := json.Unmarshal(ch.Message.Content, &s); err == nil && s != "" {
			blocks = append(blocks, anthropic.ContentBlock{Type: "text", Text: s})
		}
	}
	for _, tc := range ch.Message.ToolCalls {
		if tc.Type != "" && tc.Type != "function" {
			continue
		}
		args := json.RawMessage(tc.Function.Arguments)
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		blocks = append(blocks, anthropic.ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: args,
		})
	}
	stop := normalizeFinishReason(ch.FinishReason)
	return &TurnResult{
		Blocks:       blocks,
		StopReason:   stop,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

func normalizeFinishReason(fr string) string {
	switch fr {
	case "tool_calls":
		return "tool_use"
	case "stop":
		return "end_turn"
	default:
		if fr == "" {
			return "end_turn"
		}
		return fr
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
