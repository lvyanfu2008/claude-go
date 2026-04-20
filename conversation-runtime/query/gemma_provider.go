package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"goc/ccb-engine/gemma"
	"goc/ccb-engine/toolsearchwire"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/types"
)

// UseGemmaProvider checks if Gemma model should be used
func UseGemmaProvider() bool {
	if envTruthy("CLAUDE_CODE_USE_GEMMA") {
		return true
	}
	model := strings.TrimSpace(os.Getenv("CCB_ENGINE_MODEL"))
	if strings.Contains(strings.ToLower(model), "gemma") {
		return true
	}
	return false
}

// runGemmaStreamingParityModelLoop runs Gemma model via Vertex AI
func runGemmaStreamingParityModelLoop(
	ctx context.Context,
	params QueryParams,
	work []types.Message,
	in *CallModelInput,
	deps *QueryDeps,
	yield func(QueryYield, error) bool,
) error {
	if deps == nil {
		return fmt.Errorf("query: nil deps")
	}

	// Get Gemma configuration from environment
	projectID := strings.TrimSpace(os.Getenv("VERTEX_AI_PROJECT_ID"))
	location := strings.TrimSpace(os.Getenv("VERTEX_AI_LOCATION"))
	endpointID := strings.TrimSpace(os.Getenv("VERTEX_AI_ENDPOINT_ID"))
	modelName := strings.TrimSpace(os.Getenv("VERTEX_AI_MODEL_NAME"))
	
	// Use default config if not set
	config := gemma.DefaultConfig()
	if projectID != "" {
		config.ProjectID = projectID
	}
	if location != "" {
		config.Location = location
	}
	if endpointID != "" {
		config.EndpointID = endpointID
	}
	if modelName != "" {
		config.ModelName = modelName
	}

	// Create Gemma client
	client := gemma.NewClient(config)

	const maxRounds = 200
	cur := append([]types.Message(nil), work...)

	for round := 0; round < maxRounds; round++ {
		msgsJSON, err := ccbhydrate.MessagesJSONNormalized(cur, nil, messagesapi.OptionsFromEnv())
		if err != nil {
			return err
		}
		var msgsWire []gemma.Message
		if err := json.Unmarshal(msgsJSON, &msgsWire); err != nil {
			return fmt.Errorf("messages wire: %w", err)
		}

		toolsForWire := in.Tools
		if len(in.Tools) > 0 {
			if wired, errW := toolsearchwire.WireToolsJSON(in.Tools, config.ModelName, false, false, msgsJSON); errW == nil {
				toolsForWire = wired
			}
		}

		// Prepare Gemma request
		req := gemma.ChatRequest{
			Model:       config.ModelName,
			Messages:    msgsWire,
			MaxTokens:   4096,
			Temperature: 0.7,
		}

		// Add system prompt if available
		if sys := strings.TrimSpace(strings.Join([]string(in.SystemPrompt), "\n\n")); sys != "" {
			// Add system prompt as a message
			req.Messages = append([]gemma.Message{
				{
					Role:    "system",
					Content: sys,
				},
			}, req.Messages...)
		}

		// Add tools if available
		if len(toolsForWire) > 0 {
			var tools []gemma.Tool
			if err := json.Unmarshal(toolsForWire, &tools); err == nil {
				req.Tools = tools
			}
		}

		// Call Gemma
		resp, err := client.ChatCompletion(ctx, req)
		if err != nil {
			return fmt.Errorf("Gemma call failed: %w", err)
		}

		// Process the response
		if err := processGemmaResponse(ctx, deps, params, resp, cur, yield); err != nil {
			return err
		}

		// Check if we should continue
		if round >= maxRounds-1 {
			break
		}
	}

	return nil
}

// processGemmaResponse processes Gemma response and yields results
func processGemmaResponse(ctx context.Context, deps *QueryDeps, params QueryParams, resp *gemma.ChatResponse, cur []types.Message, yield func(QueryYield, error) bool) error {
	if len(resp.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content

	// Create inner message content
	inner, err := json.Marshal(map[string]interface{}{
		"role":    "assistant",
		"content": content,
	})
	if err != nil {
		return fmt.Errorf("marshal inner message: %w", err)
	}

	// Create assistant message
	assistantMsg := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Message: inner,
	}

	// Sync message ID
	types.SyncAssistantMessageID(&assistantMsg)

	// Yield assistant message
	if !yieldStreamingParity(ctx, deps, QueryYield{Message: &assistantMsg}, yield) {
		return nil
	}

	return nil
}