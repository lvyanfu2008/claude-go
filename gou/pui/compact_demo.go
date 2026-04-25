package pui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"goc/anthropicmessages"
	"goc/compactservice"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/modelenv"
	"goc/sessiontranscript"
	"goc/types"
)

// handleCompactCommand compacts the conversation using the Anthropic API.
// It calls compactservice.CompactConversation and replaces store messages
// with the compaction result (boundary, summary, attachments, hooks).
func handleCompactCommand(store *conversation.Store) (*processuserinput.ProcessUserInputBaseResult, error) {
	if store == nil || len(store.Messages) < 2 {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("Not enough messages to compact. Continue the conversation first.")},
			ShouldQuery: false,
		}, nil
	}

	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
	}
	if apiKey == "" {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("Cannot compact: ANTHROPIC_API_KEY is not set. Set it in your environment or ~/.claude/settings.json.")},
			ShouldQuery: false,
		}, nil
	}

	baseURL := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	model := strings.TrimSpace(os.Getenv("CLAUDE_MODEL"))
	if model == "" {
		model = modelenv.EffectiveMainLoopModel()
	}
	if model == "" {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("Cannot compact: no model configured. Set CLAUDE_MODEL.")},
			ShouldQuery: false,
		}, nil
	}

	summarizer := buildCompactSummarizer(apiKey, baseURL, model)

	deps := compactservice.Deps{
		Summarize: summarizer,
	}

	ctx := context.Background()
	result, err := compactservice.CompactConversation(ctx, store.Messages, deps, compactservice.CompactOptions{
		Model: model,
	})
	if err != nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice(fmt.Sprintf("Compaction failed: %v", err))},
			ShouldQuery: false,
		}, nil
	}

	postMsgs := compactservice.BuildPostCompactMessages(result)
	store.Messages = postMsgs
	store.ClearStreaming()
	store.StreamingToolUses = nil

	display := "Conversation compacted."
	if result.UserDisplayMessage != "" {
		display += " " + result.UserDisplayMessage
	}

	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{SystemNotice(display)},
		ShouldQuery: false,
	}, nil
}

// buildCompactSummarizer returns a compactservice.SummarizerFn that calls
// the Anthropic Messages API. Mirrors summarizeAutocompactAnthropic from
// conversation-runtime/query but is self-contained for the gou-demo TUI.
func buildCompactSummarizer(apiKey, baseURL, model string) compactservice.SummarizerFn {
	return func(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
		wireMsgs := append([]types.Message{}, in.Messages...)
		wireMsgs = append(wireMsgs, in.SummaryRequest)
		innerMsgs, err := messagesToWireShape(wireMsgs)
		if err != nil {
			return compactservice.SummaryStreamResult{}, fmt.Errorf("compact: wire messages: %w", err)
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
		if sys != "" {
			req["system"] = sys
		}

		body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
		if err != nil {
			return compactservice.SummaryStreamResult{}, err
		}

		acc := &compactStreamAccumulator{}
		err = anthropicmessages.PostStream(ctx, anthropicmessages.PostStreamParams{
			BaseURL: baseURL,
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

		summary := acc.Text()
		if summary == "" {
			return compactservice.SummaryStreamResult{}, fmt.Errorf("compact: empty summary response")
		}

		inner := map[string]any{
			"role":    "assistant",
			"content": summary,
		}
		innerJSON, err := json.Marshal(inner)
		if err != nil {
			return compactservice.SummaryStreamResult{}, err
		}

		asst := types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    sessiontranscript.NewUUID(),
			Message: innerJSON,
		}
		types.SyncAssistantMessageID(&asst)

		return compactservice.SummaryStreamResult{
			AssistantMessage: asst,
		}, nil
	}
}

// compactStreamAccumulator accumulates text from an Anthropic Messages SSE stream.
type compactStreamAccumulator struct {
	text strings.Builder
}

// OnEvent handles a single MessageStreamEvent from the SSE stream.
// Only text_delta content blocks are accumulated; all other events are skipped.
func (a *compactStreamAccumulator) OnEvent(ev anthropicmessages.MessageStreamEvent) error {
	if ev.Type != "content_block_delta" {
		return nil
	}
	var body struct {
		Index int `json:"index"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(ev.Raw, &body); err != nil {
		return nil
	}
	if body.Delta.Type == "text_delta" {
		a.text.WriteString(body.Delta.Text)
	}
	return nil
}

// Text returns the accumulated text from the stream.
func (a *compactStreamAccumulator) Text() string {
	return a.text.String()
}

// messagesToWireShape converts Message types to API wire format (role/content pairs).
// Mirrors wireShapeFromMessages in conversation-runtime/query/autocompact_adapter.go.
func messagesToWireShape(messages []types.Message) ([]any, error) {
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		if m.Type != types.MessageTypeUser && m.Type != types.MessageTypeAssistant {
			continue
		}
		if len(m.Message) == 0 {
			continue
		}
		var envelope map[string]any
		if err := json.Unmarshal(m.Message, &envelope); err != nil {
			return nil, err
		}
		out = append(out, envelope)
	}
	return out, nil
}
