package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"goc/anthropicmessages"
)

// geminiBaseURL returns the Gemini API base URL.
func geminiBaseURL() string {
	if b := strings.TrimSpace(os.Getenv("GEMINI_BASE_URL")); b != "" {
		return strings.TrimSuffix(b, "/")
	}
	return "https://generativelanguage.googleapis.com/v1beta"
}

// geminiAPIKey returns the Gemini API key.
func geminiAPIKey() string {
	return strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
}

// geminiModelPath normalizes a model name to a Gemini API path.
func geminiModelPath(model string) string {
	model = strings.TrimPrefix(model, "/")
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}

// GeminiGenerateContentRequest is a simplified Gemini API request.
type GeminiGenerateContentRequest struct {
	Contents          []GeminiContent    `json:"contents"`
	SystemInstruction *GeminiInstruction `json:"systemInstruction,omitempty"`
	Tools             []GeminiTool       `json:"tools,omitempty"`
	GenerationConfig  *GeminiGenConfig   `json:"generationConfig,omitempty"`
}

// GeminiContent represents a conversation turn.
type GeminiContent struct {
	Role  string        `json:"role"`
	Parts []GeminiPart  `json:"parts"`
}

// GeminiPart is a single content part (text, functionCall, functionResponse).
type GeminiPart struct {
	Text             string                 `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall    `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

// GeminiFunctionCall represents a model function call.
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// GeminiFunctionResponse represents a function result.
type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiInstruction holds system instructions.
type GeminiInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiTool wraps function declarations.
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

// GeminiFunctionDeclaration declares a function the model can call.
type GeminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// GeminiGenConfig is optional generation configuration.
type GeminiGenConfig struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiStreamChunk is a single SSE data payload from Gemini.
type GeminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Role  string        `json:"role"`
			Parts []GeminiPart  `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// postGeminiStream sends a Gemini request and reads SSE chunks.
func postGeminiStream(ctx context.Context, model string, body GeminiGenerateContentRequest, emit func(GeminiStreamChunk) error) error {
	url := geminiBaseURL() + "/" + geminiModelPath(model) + ":streamGenerateContent?alt=sse"

	reqBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("gemini marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", geminiAPIKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("gemini post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("gemini api %s: %s", resp.Status, string(b))
	}

	return anthropicmessages.ReadSSE(resp.Body, func(data []byte) error {
		if len(data) == 0 || string(data) == "[DONE]" {
			return nil
		}
		var chunk GeminiStreamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			return fmt.Errorf("gemini sse parse: %w", err)
		}
		return emit(chunk)
	})
}
