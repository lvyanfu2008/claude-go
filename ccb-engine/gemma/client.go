package gemma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

// Config holds Vertex AI configuration
type Config struct {
	ProjectID       string
	Location        string
	EndpointID      string
	ModelName       string
	DedicatedDomain string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		ProjectID:  "gen-lang-client-0234055272",
		Location:   "us-central1",
		EndpointID: "mg-endpoint-de951b0c-003e-4d90-bc77-221a137c655a",
		ModelName:  "gemma-7b",

		DedicatedDomain: "mg-endpoint-de951b0c-003e-4d90-bc77-221a137c655a.us-central1-584042383597.prediction.vertexai.goog",
	}
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Stream      bool                   `json:"stream,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool represents a tool definition
type Tool struct {
	Type     string                 `json:"type"`
	Function FunctionDefinition     `json:"function"`
	Extra    map[string]interface{} `json:"-"`
}

// FunctionDefinition defines a function tool
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// VertexAIRequest represents Vertex AI rawPredict request
type VertexAIRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// VertexAIResponse represents Vertex AI rawPredict response
type VertexAIResponse struct {
	Predictions []struct {
		Content string `json:"content"`
	} `json:"predictions"`
}

// Client represents a Gemma client
type Client struct {
	config Config
	client *http.Client
}

// NewClient creates a new Gemma client
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// ChatCompletion sends a chat completion request to Vertex AI
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert to Vertex AI request
	vertexReq, err := c.convertToVertexAIRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert to Vertex AI request: %w", err)
	}

	// Call Vertex AI
	vertexResp, err := c.callVertexAI(ctx, vertexReq)
	if err != nil {
		return nil, fmt.Errorf("Vertex AI call failed: %w", err)
	}

	// Convert to OpenAI-compatible response
	return c.convertToOpenAIResponse(vertexResp, req), nil
}

// convertToVertexAIRequest converts OpenAI-compatible request to Vertex AI format
func (c *Client) convertToVertexAIRequest(req ChatRequest) (VertexAIRequest, error) {
	// Convert messages to prompt string
	var promptBuilder strings.Builder
	for _, msg := range req.Messages {
		promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	promptBuilder.WriteString("assistant: ")

	return VertexAIRequest{
		Prompt:      promptBuilder.String(),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}, nil
}

// getVertexAIToken gets authentication token for Vertex AI
func (c *Client) getVertexAIToken(ctx context.Context) (string, error) {
	credentials, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("获取凭证失败: %v", err)
	}

	token, err := credentials.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("获取 Token 失败: %v", err)
	}

	return token.AccessToken, nil
}

// callVertexAI calls Vertex AI rawPredict endpoint
func (c *Client) callVertexAI(ctx context.Context, req VertexAIRequest) (*VertexAIResponse, error) {
	token, err := c.getVertexAIToken(ctx)
	if err != nil {
		return nil, err
	}

	// 注意这里结尾是 :rawPredict 而不是 :predict
	url := fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/endpoints/%s:rawPredict",
		c.config.DedicatedDomain, c.config.ProjectID, c.config.Location, c.config.EndpointID)

	jsonPayload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %v", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Vertex AI 返回错误: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	var vertexResp VertexAIResponse
	if err := json.Unmarshal(body, &vertexResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &vertexResp, nil
}

// convertToOpenAIResponse converts Vertex AI response to OpenAI response
func (c *Client) convertToOpenAIResponse(vertexResp *VertexAIResponse, req ChatRequest) *ChatResponse {
	content := ""
	if len(vertexResp.Predictions) > 0 {
		content = vertexResp.Predictions[0].Content
	}

	// Simple token estimation (in production, use proper tokenizer)
	promptTokens := len(req.Messages) * 10
	completionTokens := len(content) / 4

	return &ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

// Simple helper function for direct usage
func ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	client := NewClient(DefaultConfig())
	return client.ChatCompletion(ctx, req)
}
