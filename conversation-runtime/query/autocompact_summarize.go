package query

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	goccontext "goc/context"
	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
	"goc/ccb-engine/gemma"
	"goc/compactservice"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/types"
)

// autocompactOpenAIMaxWire mirrors TS compact streaming maxOutputTokensOverride:
// Math.min(COMPACT_MAX_OUTPUT_TOKENS, getMaxOutputTokensForModel(model)), then
// [ClampOpenAICompatibleMaxTokens] (CLAUDE_CODE_OPENAI_MAX_OUTPUT_TOKENS_CAP, default 8192).
func autocompactOpenAIMaxWire(in compactservice.SummaryStreamInput) int {
	m := strings.TrimSpace(in.Model)
	req := in.MaxOutputTokens
	if req <= 0 {
		req = compactservice.CompactMaxOutputTokens
	}
	modelCap := goccontext.GetMaxOutputTokensForModel(m)
	if req > modelCap {
		req = modelCap
	}
	return ClampOpenAICompatibleMaxTokens(req)
}

// summarizeAutocompact mirrors TS [queryModel] routing for a single text-only compact
// summary call, in the same order as [queryLoop] streaming parity:
// Gemma → OpenAI non-stream → OpenAI SSE → Anthropic Messages.
func summarizeAutocompact(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	openAI := StreamingUsesOpenAIChat()
	openAINoStream := openAI && OpenAIChatNoStreamEnabled()
	switch {
	case UseGemmaProvider():
		return summarizeAutocompactGemma(ctx, in)
	case openAINoStream:
		return summarizeAutocompactOpenAINoStream(ctx, in)
	case openAI:
		return summarizeAutocompactOpenAIStream(ctx, in)
	default:
		return summarizeAutocompactAnthropic(ctx, in)
	}
}

func summarizeAutocompactAnthropic(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
	}
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if base == "" {
		base = "https://api.anthropic.com"
	}
	if apiKey == "" {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact: ANTHROPIC_API_KEY missing — cannot summarize")
	}

	model := strings.TrimSpace(in.Model)
	if model == "" {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact: model missing")
	}

	wireMsgs := append([]types.Message{}, in.Messages...)
	wireMsgs = append(wireMsgs, in.SummaryRequest)
	innerMsgs, err := wireShapeFromMessages(wireMsgs)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact wire msgs: %w", err)
	}

	sys := strings.TrimSpace(strings.Join(in.SystemPrompt, "\n\n"))
	maxOut := in.MaxOutputTokens
	if maxOut <= 0 {
		maxOut = compactservice.CompactMaxOutputTokens
	}
	req := map[string]any{
		"model":      model,
		"max_tokens": maxOut,
		"messages":   innerMsgs,
		"stream":     true,
		"thinking":   map[string]any{"type": "disabled"},
	}
	if sys != "" {
		req["system"] = sys
	}
	body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	acc := newAssistantStreamAccumulator()
	err = anthropicmessages.PostStream(ctx, anthropicmessages.PostStreamParams{
		BaseURL: base,
		APIKey:  apiKey,
		Body:    body,
		HTTP:    http.DefaultClient,
		Emit: func(ev anthropicmessages.MessageStreamEvent) error {
			return acc.OnEvent(ev)
		},
	})
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	uuid := randomUUID()
	inner, err := acc.AssistantWire(uuid)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
	}
	types.SyncAssistantMessageID(&asst)

	usage := compactservice.GetTokenUsage(asst)
	return compactservice.SummaryStreamResult{AssistantMessage: asst, Usage: usage}, nil
}

func summarizeAutocompactOpenAIStream(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact: OPENAI_API_KEY missing — cannot summarize (openai provider)")
	}
	base := openAIBaseURLFromEnv()
	model := ResolveOpenAIModel(strings.TrimSpace(in.Model))
	maxOut := autocompactOpenAIMaxWire(in)

	wireMsgs := append([]types.Message{}, in.Messages...)
	wireMsgs = append(wireMsgs, in.SummaryRequest)
	msgsJSON, err := ccbhydrate.MessagesJSONNormalized(wireMsgs, nil, messagesapi.OptionsFromEnv())
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact openai hydrate: %w", err)
	}
	openaiMsgs, err := anthropicWireMessagesToOpenAI(msgsJSON, in.SystemPrompt, model)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact openai wire: %w", err)
	}

	req := map[string]any{
		"model":    model,
		"messages": openaiMsgs,
		"stream":   true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
		"max_tokens": maxOut,
	}
	body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	acc := newAssistantStreamAccumulator()
	if err := PostOpenAIChatStream(ctx, OpenAIPostStreamParams{
		BaseURL: base,
		APIKey:  apiKey,
		Body:    body,
		HTTP:    http.DefaultClient,
		Emit:    acc.OnEvent,
	}); err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	uuid := randomUUID()
	inner, err := acc.AssistantWire(uuid)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
	}
	types.SyncAssistantMessageID(&asst)
	usage := compactservice.GetTokenUsage(asst)
	return compactservice.SummaryStreamResult{AssistantMessage: asst, Usage: usage}, nil
}

func summarizeAutocompactOpenAINoStream(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact: OPENAI_API_KEY missing — cannot summarize (openai provider)")
	}
	base := strings.TrimSpace(openAIBaseURLFromEnv())
	model := ResolveOpenAIModel(strings.TrimSpace(in.Model))
	maxOut := autocompactOpenAIMaxWire(in)
	url := strings.TrimSuffix(base, "/") + "/chat/completions"

	wireMsgs := append([]types.Message{}, in.Messages...)
	wireMsgs = append(wireMsgs, in.SummaryRequest)
	msgsJSON, err := ccbhydrate.MessagesJSONNormalized(wireMsgs, nil, messagesapi.OptionsFromEnv())
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact openai hydrate: %w", err)
	}
	openaiMsgs, err := anthropicWireMessagesToOpenAI(msgsJSON, in.SystemPrompt, model)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact openai wire: %w", err)
	}

	req := map[string]any{
		"model":      model,
		"messages":   openaiMsgs,
		"max_tokens": maxOut,
	}
	body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	if apilog.ApiBodyLoggingEnabled() {
		apilog.PrepareIfEnabled()
	}
	apilog.LogRequestBody("POST "+url+" (autocompact no-stream)", body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	httpReq.Header.Set("authorization", "Bearer "+apiKey)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	respBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	apilog.LogResponseBody("POST "+url+" (autocompact no-stream "+resp.Status+")", respBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact openai chat %s: %s", resp.Status, truncateOpenAIErr(string(respBody), 800))
	}

	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAINonStreamChatResponse(respBody, model, acc.OnEvent); err != nil {
		return compactservice.SummaryStreamResult{}, err
	}

	uuid := randomUUID()
	inner, err := acc.AssistantWire(uuid)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
	}
	types.SyncAssistantMessageID(&asst)
	usage := compactservice.GetTokenUsage(asst)
	return compactservice.SummaryStreamResult{AssistantMessage: asst, Usage: usage}, nil
}

func summarizeAutocompactGemma(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	projectID := strings.TrimSpace(os.Getenv("VERTEX_AI_PROJECT_ID"))
	location := strings.TrimSpace(os.Getenv("VERTEX_AI_LOCATION"))
	endpointID := strings.TrimSpace(os.Getenv("VERTEX_AI_ENDPOINT_ID"))
	modelName := strings.TrimSpace(os.Getenv("VERTEX_AI_MODEL_NAME"))

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

	client := gemma.NewClient(config)

	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = config.ModelName
	}
	maxOut := in.MaxOutputTokens
	if maxOut <= 0 {
		maxOut = compactservice.CompactMaxOutputTokens
	}

	wireMsgs := append([]types.Message{}, in.Messages...)
	wireMsgs = append(wireMsgs, in.SummaryRequest)
	msgsJSON, err := ccbhydrate.MessagesJSONNormalized(wireMsgs, nil, messagesapi.OptionsFromEnv())
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact gemma hydrate: %w", err)
	}
	msgsWire, err := gemmaWireMessagesFromHydratedJSON(msgsJSON)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact gemma wire: %w", err)
	}

	req := gemma.ChatRequest{
		Model:       config.ModelName,
		Messages:    msgsWire,
		MaxTokens:   maxOut,
		Temperature: 0,
	}
	sys := strings.TrimSpace(strings.Join(in.SystemPrompt, "\n\n"))
	if sys != "" {
		req.Messages = append([]gemma.Message{{Role: "system", Content: sys}}, req.Messages...)
	}

	raw, err := client.ChatCompletionRaw(ctx, req)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact gemma: %w", err)
	}
	innerBody, err := gemma.UnwrapVertexPredictionsOpenAI(raw)
	if err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact gemma unwrap: %w", err)
	}
	innerBody = NormalizeOpenAINonStreamChatBodyToolCallsLoose(innerBody)

	acc := newAssistantStreamAccumulator()
	if err := ReplayOpenAINonStreamChatResponse(innerBody, model, acc.OnEvent); err != nil {
		return compactservice.SummaryStreamResult{}, fmt.Errorf("autocompact gemma replay: %w", err)
	}

	uuid := randomUUID()
	inner, err := acc.AssistantWire(uuid)
	if err != nil {
		return compactservice.SummaryStreamResult{}, err
	}
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
	}
	types.SyncAssistantMessageID(&asst)
	usage := compactservice.GetTokenUsage(asst)
	return compactservice.SummaryStreamResult{AssistantMessage: asst, Usage: usage}, nil
}
