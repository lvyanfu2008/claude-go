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

	"goc/ccb-engine/apilog"

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
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	// ToolsJSON is an OpenAI-style tools JSON array (verbatim) when Tools is empty or not used; not sent in JSON.Marshal(ChatRequest).
	ToolsJSON  json.RawMessage        `json:"-"`
	ToolChoice interface{}            `json:"tool_choice,omitempty"`
	Extra      map[string]interface{} `json:"-"`
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

// --- Vertex :rawPredict wire (chatCompletions) ---

type vertexChatContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type vertexChatMessage struct {
	Role    string                  `json:"role"`
	Content []vertexChatContentPart `json:"content"`
}

// vertexToolWire matches OpenAI chat.completions "tools" entries (also what Vertex @requestFormat=chatCompletions expects).
type vertexToolWire struct {
	Type     string             `json:"type"`
	Function vertexFunctionWire `json:"function"`
}

type vertexFunctionWire struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type vertexInstance struct {
	RequestFormat string              `json:"@requestFormat"`
	Messages      []vertexChatMessage `json:"messages"`
	Tools         json.RawMessage     `json:"tools,omitempty"`
	ToolChoice    any                 `json:"tool_choice,omitempty"`
	MaxTokens     int                 `json:"max_tokens,omitempty"`
	Temperature   float64             `json:"temperature,omitempty"`
	TopP          float64             `json:"top_p,omitempty"`
}

type vertexPredictRequest struct {
	Instances []vertexInstance `json:"instances"`
}

// vertexPrediction unmarshals legacy prediction array elements (string or {"content":…}).
type vertexPrediction struct {
	Content string
}

func (p *vertexPrediction) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("prediction as string: %w", err)
		}
		p.Content = s
		return nil
	}
	var obj struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("prediction as object: %w", err)
	}
	p.Content = obj.Content
	return nil
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
	wire, err := c.buildVertexPredictRequest(req)
	if err != nil {
		return nil, fmt.Errorf("build Vertex request: %w", err)
	}
	body, err := c.doVertexRawPredict(ctx, wire)
	if err != nil {
		return nil, err
	}
	return parseVertexPredictResponse(body, req)
}

// ChatCompletionRaw returns the full Vertex :rawPredict HTTP body on success (includes {"predictions":…}).
func (c *Client) ChatCompletionRaw(ctx context.Context, req ChatRequest) ([]byte, error) {
	wire, err := c.buildVertexPredictRequest(req)
	if err != nil {
		return nil, fmt.Errorf("build Vertex request: %w", err)
	}
	return c.doVertexRawPredict(ctx, wire)
}

// UnwrapVertexPredictionsOpenAI returns the inner OpenAI chat.completion JSON when the envelope is
// {"predictions":{...}} with predictions as an object (not a legacy string/array).
func UnwrapVertexPredictionsOpenAI(vertexFullBody []byte) ([]byte, error) {
	var env struct {
		Predictions json.RawMessage `json:"predictions"`
	}
	if err := json.Unmarshal(vertexFullBody, &env); err != nil {
		return nil, fmt.Errorf("vertex envelope: %w", err)
	}
	if len(env.Predictions) == 0 || env.Predictions[0] != '{' {
		return nil, fmt.Errorf("vertex predictions: missing or non-object")
	}
	return env.Predictions, nil
}

func (c *Client) buildVertexPredictRequest(req ChatRequest) (*vertexPredictRequest, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages: empty")
	}
	msgs := make([]vertexChatMessage, 0, len(req.Messages)+1)
	for _, m := range req.Messages {
		msgs = append(msgs, vertexChatMessage{
			Role: m.Role,
			Content: []vertexChatContentPart{
				{Type: "text", Text: m.Content},
			},
		})
	}

	var toolsRaw json.RawMessage
	if len(req.Tools) > 0 {
		wireTools := toolsToVertexWire(req.Tools)
		if len(wireTools) > 0 {
			b, err := json.Marshal(wireTools)
			if err != nil {
				return nil, fmt.Errorf("tools: %w", err)
			}
			toolsRaw = b
		}
	} else if len(req.ToolsJSON) > 0 {
		toolsRaw = append(json.RawMessage(nil), req.ToolsJSON...)
	}
	if len(toolsRaw) > 0 {
		insertVertexToolsRoleMessage(&msgs, string(toolsRaw))
	}

	inst := vertexInstance{
		RequestFormat: "chatCompletions",
		Messages:      msgs,
	}
	if req.MaxTokens > 0 {
		inst.MaxTokens = req.MaxTokens
	}
	if req.Temperature != 0 {
		inst.Temperature = req.Temperature
	}
	if req.TopP != 0 {
		inst.TopP = req.TopP
	}
	if len(toolsRaw) > 0 {
		inst.Tools = toolsRaw
	}
	if req.ToolChoice != nil {
		inst.ToolChoice = req.ToolChoice
	}
	return &vertexPredictRequest{Instances: []vertexInstance{inst}}, nil
}

// insertVertexToolsRoleMessage inserts {"role":"tools","content":[{"type":"text","text":...}]}
// after any leading system messages so the model sees tools as their own role in messages[].
func insertVertexToolsRoleMessage(msgs *[]vertexChatMessage, toolsJSON string) {
	if msgs == nil || len(toolsJSON) == 0 {
		return
	}
	arr := *msgs
	i := 0
	for i < len(arr) && strings.EqualFold(arr[i].Role, "system") {
		i++
	}
	toolsMsg := vertexChatMessage{
		Role: "tools",
		Content: []vertexChatContentPart{
			{Type: "text", Text: toolsJSON},
		},
	}
	out := make([]vertexChatMessage, 0, len(arr)+1)
	out = append(out, arr[:i]...)
	out = append(out, toolsMsg)
	out = append(out, arr[i:]...)
	*msgs = out
}

func toolsToVertexWire(tools []Tool) []vertexToolWire {
	out := make([]vertexToolWire, 0, len(tools))
	for _, t := range tools {
		name := strings.TrimSpace(t.Function.Name)
		if name == "" {
			continue
		}
		typ := strings.TrimSpace(t.Type)
		if typ == "" {
			typ = "function"
		}
		fn := vertexFunctionWire{
			Name:        name,
			Description: strings.TrimSpace(t.Function.Description),
		}
		if len(t.Function.Parameters) > 0 {
			fn.Parameters = t.Function.Parameters
		}
		out = append(out, vertexToolWire{Type: typ, Function: fn})
	}
	return out
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

// doVertexRawPredict POSTs wire to Vertex :rawPredict and returns the response body on HTTP 200.
func (c *Client) doVertexRawPredict(ctx context.Context, wire *vertexPredictRequest) ([]byte, error) {
	token, err := c.getVertexAIToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/endpoints/%s:rawPredict",
		c.config.DedicatedDomain, c.config.ProjectID, c.config.Location, c.config.EndpointID)

	jsonPayload, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	if apilog.ApiBodyLoggingEnabled() {
		apilog.PrepareIfEnabled()
	}
	apilog.LogRequestBody("POST "+url+" (vertex-rawPredict)", jsonPayload)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}
	apilog.LogResponseBody("POST "+url+" (vertex-rawPredict "+resp.Status+")", body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Vertex AI 返回错误: %d - %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func parseVertexPredictResponse(body []byte, req ChatRequest) (*ChatResponse, error) {
	var env struct {
		Predictions json.RawMessage `json:"predictions"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("envelope: %w", err)
	}
	if len(env.Predictions) == 0 {
		return nil, fmt.Errorf("missing predictions")
	}

	switch env.Predictions[0] {
	case '{':
		var out ChatResponse
		if err := json.Unmarshal(env.Predictions, &out); err != nil {
			return nil, fmt.Errorf("predictions object: %w", err)
		}
		if strings.TrimSpace(out.Model) == "" {
			out.Model = req.Model
		}
		if out.Object == "" {
			out.Object = "chat.completion"
		}
		return &out, nil
	case '[':
		var parts []vertexPrediction
		if err := json.Unmarshal(env.Predictions, &parts); err != nil {
			return nil, fmt.Errorf("predictions array: %w", err)
		}
		return legacyPredictionsArrayToChatResponse(parts, req), nil
	default:
		return nil, fmt.Errorf("predictions: unexpected JSON")
	}
}

func legacyPredictionsArrayToChatResponse(parts []vertexPrediction, req ChatRequest) *ChatResponse {
	content := ""
	if len(parts) > 0 {
		content = parts[0].Content
	}
	promptTokens := len(req.Messages) * 10
	completionTokens := len(content) / 4
	if len(content) == 0 {
		completionTokens = 0
	}
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
