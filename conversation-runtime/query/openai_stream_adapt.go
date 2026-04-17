package query

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"goc/anthropicmessages"
)

// openAIStreamAdapter mirrors src/api-client/openai/streamAdapter.ts adaptOpenAIStreamToAnthropic.
type openAIStreamAdapter struct {
	model string

	started bool
	msgID   string

	currentContentIndex int
	toolBlocks          map[int]*openAIToolBlockState

	thinkingBlockOpen bool
	textBlockOpen     bool

	inputTokens  int
	outputTokens int
	cachedTokens int

	openBlockIndices map[int]struct{}
}

type openAIToolBlockState struct {
	contentIndex int
	id           string
	name         string
	// Some OpenAI-compatible APIs send function.arguments as a JSON object/array instead of a string.
	// We marshal that once to a string and emit a single input_json_delta (OpenAI spec uses string fragments).
	emittedStructuredArgs bool
}

func newOpenAIStreamAdapter(model string) *openAIStreamAdapter {
	return &openAIStreamAdapter{
		model:            model,
		currentContentIndex: -1,
		toolBlocks:       make(map[int]*openAIToolBlockState),
		openBlockIndices: make(map[int]struct{}),
		msgID:            openAIMessageStreamID(),
	}
}

func openAIMessageStreamID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "msg_" + hex.EncodeToString(b[:])[:24]
}

func openAIToolPlaceholderID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "toolu_" + hex.EncodeToString(b[:])[:24]
}

// openAIArgumentsFragment returns the next fragment to append to tool JSON input.
// OpenAI's schema uses a string for function.arguments; some proxies send a decoded object instead.
func openAIArgumentsFragment(fn map[string]any, st *openAIToolBlockState) (frag string, ok bool) {
	raw, exists := fn["arguments"]
	if !exists || raw == nil {
		return "", false
	}
	if s, okStr := raw.(string); okStr {
		return s, true
	}
	if st.emittedStructuredArgs {
		return "", false
	}
	st.emittedStructuredArgs = true
	b, err := json.Marshal(raw)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return "", false
	}
	return string(b), true
}

func (a *openAIStreamAdapter) markOpen(idx int) {
	a.openBlockIndices[idx] = struct{}{}
}

func (a *openAIStreamAdapter) markClosed(idx int) {
	delete(a.openBlockIndices, idx)
}

func emitStreamObj(obj map[string]any, emit func(anthropicmessages.MessageStreamEvent) error) error {
	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var ev anthropicmessages.MessageStreamEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	return emit(ev)
}

func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "end_turn"
	default:
		return "end_turn"
	}
}

func (a *openAIStreamAdapter) applyUsageFromChunk(raw json.RawMessage) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	var u struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		PromptTokensDetails     *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		return
	}
	if u.PromptTokens > 0 {
		a.inputTokens = u.PromptTokens
	}
	if u.CompletionTokens > 0 {
		a.outputTokens = u.CompletionTokens
	}
	if u.PromptTokensDetails != nil && u.PromptTokensDetails.CachedTokens > 0 {
		a.cachedTokens = u.PromptTokensDetails.CachedTokens
	}
}

func (a *openAIStreamAdapter) HandleChunk(chunkJSON []byte, emit func(anthropicmessages.MessageStreamEvent) error) error {
	var chunk struct {
		Choices []struct {
			Delta        json.RawMessage `json:"delta"`
			FinishReason string          `json:"finish_reason"`
		} `json:"choices"`
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(chunkJSON, &chunk); err != nil {
		return fmt.Errorf("openai chunk: %w", err)
	}

	a.applyUsageFromChunk(chunk.Usage)

	if !a.started {
		a.started = true
		if err := emitStreamObj(map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":            a.msgID,
				"type":          "message",
				"role":          "assistant",
				"content":       []any{},
				"model":         a.model,
				"stop_reason":   nil,
				"stop_sequence": nil,
				"usage": map[string]any{
					"input_tokens":                 a.inputTokens,
					"output_tokens":                0,
					"cache_creation_input_tokens":  0,
					"cache_read_input_tokens":      a.cachedTokens,
				},
			},
		}, emit); err != nil {
			return err
		}
	}

	var delta struct {
		Content          *string         `json:"content"`
		ReasoningContent *string         `json:"reasoning_content"`
		ToolCalls        json.RawMessage `json:"tool_calls"`
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	ch0 := chunk.Choices[0]
	if len(ch0.Delta) > 0 && string(ch0.Delta) != "null" {
		_ = json.Unmarshal(ch0.Delta, &delta)
	}

	// reasoning_content → thinking
	if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
		if !a.thinkingBlockOpen {
			a.currentContentIndex++
			a.thinkingBlockOpen = true
			a.markOpen(a.currentContentIndex)
			if err := emitStreamObj(map[string]any{
				"type":  "content_block_start",
				"index": a.currentContentIndex,
				"content_block": map[string]any{
					"type": "thinking", "thinking": "", "signature": "",
				},
			}, emit); err != nil {
				return err
			}
		}
		if err := emitStreamObj(map[string]any{
			"type":  "content_block_delta",
			"index": a.currentContentIndex,
			"delta": map[string]any{
				"type": "thinking_delta", "thinking": *delta.ReasoningContent,
			},
		}, emit); err != nil {
			return err
		}
	}

	if delta.Content != nil && *delta.Content != "" {
		if a.thinkingBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.thinkingBlockOpen = false
		}
		if !a.textBlockOpen {
			a.currentContentIndex++
			a.textBlockOpen = true
			a.markOpen(a.currentContentIndex)
			if err := emitStreamObj(map[string]any{
				"type":  "content_block_start",
				"index": a.currentContentIndex,
				"content_block": map[string]any{
					"type": "text", "text": "",
				},
			}, emit); err != nil {
				return err
			}
		}
		if err := emitStreamObj(map[string]any{
			"type":  "content_block_delta",
			"index": a.currentContentIndex,
			"delta": map[string]any{
				"type": "text_delta", "text": *delta.Content,
			},
		}, emit); err != nil {
			return err
		}
	}

	if len(delta.ToolCalls) > 0 && string(delta.ToolCalls) != "null" {
		var tcalls []map[string]any
		if err := json.Unmarshal(delta.ToolCalls, &tcalls); err == nil {
			for _, tc := range tcalls {
				tcIndex := 0
				if v, ok := tc["index"].(float64); ok {
					tcIndex = int(v)
				}
				if _, exists := a.toolBlocks[tcIndex]; !exists {
					if a.thinkingBlockOpen {
						if err := emitStreamObj(map[string]any{
							"type": "content_block_stop", "index": a.currentContentIndex,
						}, emit); err != nil {
							return err
						}
						a.markClosed(a.currentContentIndex)
						a.thinkingBlockOpen = false
					}
					if a.textBlockOpen {
						if err := emitStreamObj(map[string]any{
							"type": "content_block_stop", "index": a.currentContentIndex,
						}, emit); err != nil {
							return err
						}
						a.markClosed(a.currentContentIndex)
						a.textBlockOpen = false
					}
					a.currentContentIndex++
					toolID := openAIToolPlaceholderID()
					if idStr, ok := tc["id"].(string); ok && idStr != "" {
						toolID = idStr
					}
					toolName := ""
					if fn, ok := tc["function"].(map[string]any); ok {
						if n, ok := fn["name"].(string); ok {
							toolName = n
						}
					}
					a.toolBlocks[tcIndex] = &openAIToolBlockState{
						contentIndex: a.currentContentIndex,
						id:           toolID,
						name:         toolName,
					}
					a.markOpen(a.currentContentIndex)
					if err := emitStreamObj(map[string]any{
						"type":  "content_block_start",
						"index": a.currentContentIndex,
						"content_block": map[string]any{
							"type": "tool_use", "id": toolID, "name": toolName, "input": map[string]any{},
						},
					}, emit); err != nil {
						return err
					}
				}
				st := a.toolBlocks[tcIndex]
				if st == nil {
					continue
				}
				if fn, ok := tc["function"].(map[string]any); ok {
					if n, ok := fn["name"].(string); ok && n != "" {
						st.name = n
					}
					if arg, okFrag := openAIArgumentsFragment(fn, st); okFrag && arg != "" {
						if err := emitStreamObj(map[string]any{
							"type":  "content_block_delta",
							"index": st.contentIndex,
							"delta": map[string]any{
								"type": "input_json_delta", "partial_json": arg,
							},
						}, emit); err != nil {
							return err
						}
					}
				}
				if idStr, ok := tc["id"].(string); ok && idStr != "" {
					st.id = idStr
				}
			}
		}
	}

	if ch0.FinishReason != "" {
		if a.thinkingBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.thinkingBlockOpen = false
		}
		if a.textBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.textBlockOpen = false
		}
		idxSet := make(map[int]struct{})
		for _, st := range a.toolBlocks {
			if _, open := a.openBlockIndices[st.contentIndex]; open {
				idxSet[st.contentIndex] = struct{}{}
			}
		}
		toolIdxs := make([]int, 0, len(idxSet))
		for idx := range idxSet {
			toolIdxs = append(toolIdxs, idx)
		}
		sort.Ints(toolIdxs)
		for _, idx := range toolIdxs {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": idx,
			}, emit); err != nil {
				return err
			}
			a.markClosed(idx)
		}
		stop := mapFinishReason(ch0.FinishReason)
		if err := emitStreamObj(map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason":   stop,
				"stop_sequence": nil,
				"usage": map[string]any{
					"output_tokens": a.outputTokens,
				},
			},
		}, emit); err != nil {
			return err
		}
		if err := emitStreamObj(map[string]any{"type": "message_stop"}, emit); err != nil {
			return err
		}
	}
	return nil
}

func (a *openAIStreamAdapter) FlushOpenBlocks(emit func(anthropicmessages.MessageStreamEvent) error) error {
	for idx := range a.openBlockIndices {
		if err := emitStreamObj(map[string]any{
			"type": "content_block_stop", "index": idx,
		}, emit); err != nil {
			return err
		}
	}
	a.openBlockIndices = make(map[int]struct{})
	return nil
}

// ReplayOpenAINonStreamChatResponse converts a full POST /v1/chat/completions JSON body (stream:false)
// into the same Anthropic-shaped stream events [openAIStreamAdapter] would emit, so
// [assistantStreamAccumulator] and the streaming tool executor stay unchanged.
func ReplayOpenAINonStreamChatResponse(respBody []byte, model string, emit func(anthropicmessages.MessageStreamEvent) error) error {
	var head struct {
		Error   *struct{ Message string `json:"message"` } `json:"error"`
		Choices []json.RawMessage `json:"choices"`
		Usage   json.RawMessage   `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &head); err != nil {
		return fmt.Errorf("openai non-stream json: %w", err)
	}
	if head.Error != nil && strings.TrimSpace(head.Error.Message) != "" {
		return fmt.Errorf("openai non-stream api error: %s", head.Error.Message)
	}
	if len(head.Choices) == 0 {
		return fmt.Errorf("openai non-stream: empty choices")
	}
	var ch struct {
		Message struct {
			Content          json.RawMessage   `json:"content"`
			ReasoningContent *string           `json:"reasoning_content"`
			ToolCalls        []map[string]any `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}
	if err := json.Unmarshal(head.Choices[0], &ch); err != nil {
		return fmt.Errorf("openai non-stream choice: %w", err)
	}

	ad := newOpenAIStreamAdapter(model)
	kick, err := json.Marshal(map[string]any{
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{}}},
		"usage":   head.Usage,
	})
	if err != nil {
		return err
	}
	if err := ad.HandleChunk(kick, emit); err != nil {
		return err
	}

	// DeepSeek reasoner (and similar): non-stream body includes message.reasoning_content (chain-of-thought).
	// Mirror streaming deltas: emit reasoning before visible content (see HandleChunk reasoning_content branch).
	if ch.Message.ReasoningContent != nil && strings.TrimSpace(*ch.Message.ReasoningContent) != "" {
		rc, err := json.Marshal(map[string]any{
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"reasoning_content": *ch.Message.ReasoningContent},
			}},
		})
		if err != nil {
			return err
		}
		if err := ad.HandleChunk(rc, emit); err != nil {
			return err
		}
	}

	var textPieces []string
	if len(ch.Message.Content) > 0 && string(ch.Message.Content) != "null" {
		var s string
		if err := json.Unmarshal(ch.Message.Content, &s); err == nil {
			if strings.TrimSpace(s) != "" {
				textPieces = append(textPieces, s)
			}
		} else {
			var parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(ch.Message.Content, &parts); err == nil {
				for _, p := range parts {
					if p.Type == "text" && p.Text != "" {
						textPieces = append(textPieces, p.Text)
					}
				}
			}
		}
	}
	for _, piece := range textPieces {
		b, err := json.Marshal(map[string]any{
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"content": piece},
			}},
		})
		if err != nil {
			return err
		}
		if err := ad.HandleChunk(b, emit); err != nil {
			return err
		}
	}

	if len(ch.Message.ToolCalls) > 0 {
		tcalls := make([]map[string]any, 0, len(ch.Message.ToolCalls))
		for i, tc := range ch.Message.ToolCalls {
			if tc == nil {
				continue
			}
			if _, ok := tc["index"]; !ok {
				tc["index"] = float64(i)
			}
			tcalls = append(tcalls, tc)
		}
		if len(tcalls) > 0 {
			b, err := json.Marshal(map[string]any{
				"choices": []map[string]any{{
					"index": 0,
					"delta": map[string]any{"tool_calls": tcalls},
				}},
			})
			if err != nil {
				return err
			}
			if err := ad.HandleChunk(b, emit); err != nil {
				return err
			}
		}
	}

	finish := strings.TrimSpace(ch.FinishReason)
	if finish == "" {
		if len(ch.Message.ToolCalls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	endObj := map[string]any{
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": finish,
		}},
	}
	if len(head.Usage) > 0 && string(head.Usage) != "null" {
		endObj["usage"] = json.RawMessage(head.Usage)
	}
	endB, err := json.Marshal(endObj)
	if err != nil {
		return err
	}
	if err := ad.HandleChunk(endB, emit); err != nil {
		return err
	}
	return ad.FlushOpenBlocks(emit)
}
