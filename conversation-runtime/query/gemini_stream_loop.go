package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/anthropicmessages"
	"goc/messagesapi"
	"goc/types"
)

// runGeminiStreamingParityModelLoop sends requests to Gemini API and yields
// assistant messages. This is a simplified single-turn loop; full multi-turn
// tool-calling parity will follow.
func runGeminiStreamingParityModelLoop(
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
	apiKey := geminiAPIKey()
	if apiKey == "" {
		return fmt.Errorf("query gemini: set GEMINI_API_KEY")
	}

	model, err := resolveGeminiModel(strings.TrimSpace(in.ModelID))
	if err != nil {
		return fmt.Errorf("query gemini model: %w", err)
	}

	contents, _, err := anthropicMessagesToGemini(work, in.SystemPrompt, messagesapi.OptionsFromEnv())
	if err != nil {
		return fmt.Errorf("query gemini wire: %w", err)
	}

	body := GeminiGenerateContentRequest{
		Contents: contents,
	}
	if len(in.SystemPrompt) > 0 {
		allSys := ""
		for _, s := range in.SystemPrompt {
			allSys += s + "\n"
		}
		if allSys != "" {
			body.SystemInstruction = &GeminiInstruction{
				Parts: []GeminiPart{{Text: allSys}},
			}
		}
	}

	acc := newAssistantStreamAccumulator()
	ad := newGeminiStreamAdapter(model)

	if err := postGeminiStream(ctx, model, body, func(chunk GeminiStreamChunk) error {
		return ad.HandleChunk(chunk, func(ev anthropicmessages.MessageStreamEvent) error {
			return acc.OnEvent(ev)
		})
	}); err != nil {
		return fmt.Errorf("query gemini: %w", err)
	}

	inner, err := acc.AssistantWire("wire")
	if err != nil {
		return fmt.Errorf("query gemini accumulate: %w", err)
	}

	var contentExtract struct {
		Content json.RawMessage `json:"content"`
	}
	json.Unmarshal(inner, &contentExtract)
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		Message: inner,
		Content: contentExtract.Content,
	}
	types.SyncAssistantMessageID(&asst)

	yield(QueryYield{Message: &asst}, nil)
	return nil
}
