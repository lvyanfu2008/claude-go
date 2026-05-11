package query

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

	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
	"goc/compactservice"
	goccontext "goc/context"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/tstenv"
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

// isFirstPartyAnthropicBaseURL mirrors TS isFirstPartyAnthropicBaseUrl in
// src/utils/model/providers.ts: true when ANTHROPIC_BASE_URL is unset or
// points to api.anthropic.com (or api-staging.anthropic.com for ant users).
func isFirstPartyAnthropicBaseURL() bool {
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if base == "" {
		return true
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	if host == "api.anthropic.com" {
		return true
	}
	if strings.ToLower(os.Getenv("USER_TYPE")) == "ant" && host == "api-staging.anthropic.com" {
		return true
	}
	return false
}

// resolveAutocompactHaikuModel resolves a cost-effective model for autocompact
// when the main-loop model is DeepSeek. DeepSeek does not support the Anthropic
// /v1/messages endpoint, so compaction must use an OpenAI-compatible endpoint with
// a cheaper model. Precedence:
//
//	OPENAI_DEFAULT_HAIKU_MODEL > ANTHROPIC_DEFAULT_HAIKU_MODEL >
//	ResolveOpenAIModel("claude-haiku-4-5-20251001")
//
// For non-DeepSeek models, returns the main model unchanged.
func resolveAutocompactHaikuModel(mainModel string) string {
	if !tstenv.IsDeepSeekModel(mainModel) {
		return mainModel
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_DEFAULT_HAIKU_MODEL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")); v != "" {
		return v
	}
	fallback := ResolveOpenAIModel("claude-haiku-4-5-20251001")
	// If ResolveOpenAIModel returns a DeepSeek model (e.g. via CCB_ENGINE_MODEL),
	// fall back to a known safe OpenAI model.
	if tstenv.IsDeepSeekModel(fallback) {
		return "gpt-4o-mini"
	}
	return fallback
}

// summarizeAutocompact mirrors TS [queryModel] routing for a single text-only compact
// summary call, in the same order as [queryLoop] streaming parity:
// OpenAI non-stream → OpenAI SSE → Anthropic Messages.
//
// DeepSeek models are always routed to the OpenAI path because they do not support
// the Anthropic /v1/messages endpoint.
func summarizeAutocompact(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
	model := strings.TrimSpace(in.Model)
	openAI := StreamingUsesOpenAIChat() || tstenv.IsDeepSeekModel(model)
	openAINoStream := openAI && OpenAIChatNoStreamEnabled()
	switch {
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
	}
	// thinking:disabled is Anthropic-specific — third-party providers (DeepSeek)
	// that present an Anthropic-compatible /v1/messages endpoint reject it with 404.
	if isFirstPartyAnthropicBaseURL() {
		req["thinking"] = map[string]any{"type": "disabled"}
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
	var contentExtract struct {
		Content json.RawMessage `json:"content"`
	}
	json.Unmarshal(inner, &contentExtract)
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
		Content: contentExtract.Content,
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
	model := resolveAutocompactHaikuModel(strings.TrimSpace(in.Model))
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
	var contentExtract struct {
		Content json.RawMessage `json:"content"`
	}
	json.Unmarshal(inner, &contentExtract)
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
		Content: contentExtract.Content,
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
	model := resolveAutocompactHaikuModel(strings.TrimSpace(in.Model))
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
	var contentExtract struct {
		Content json.RawMessage `json:"content"`
	}
	json.Unmarshal(inner, &contentExtract)
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: inner,
		Content: contentExtract.Content,
	}
	types.SyncAssistantMessageID(&asst)
	usage := compactservice.GetTokenUsage(asst)
	return compactservice.SummaryStreamResult{AssistantMessage: asst, Usage: usage}, nil
}
