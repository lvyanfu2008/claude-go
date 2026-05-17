package query

import (
	"encoding/json"
	"fmt"
	"sync"

	"goc/anthropicmessages"
)

// geminiStreamAdapter adapts GeminiStreamChunk events into Anthropic MessageStreamEvent format.
type geminiStreamAdapter struct {
	mu        sync.Mutex
	started   bool
	textOpen  bool
	id        int64
	msgID     string
	modelID   string
}

func newGeminiStreamAdapter(model string) *geminiStreamAdapter {
	return &geminiStreamAdapter{modelID: model}
}

// makeEvent creates a MessageStreamEvent with the Type set and Raw marshaled.
func makeEvent(typ string, payload interface{}) (anthropicmessages.MessageStreamEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return anthropicmessages.MessageStreamEvent{}, err
	}
	return anthropicmessages.MessageStreamEvent{
		Type: typ,
		Raw:  raw,
	}, nil
}

// HandleChunk converts a Gemini SSE chunk into Anthropic-format events via emit.
func (a *geminiStreamAdapter) HandleChunk(chunk GeminiStreamChunk, emit func(anthropicmessages.MessageStreamEvent) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		a.started = true
		a.id++
		a.msgID = fmt.Sprintf("msg_%d", a.id)
		ev, err := makeEvent("message_start", map[string]interface{}{
			"type":    "message",
			"role":    "assistant",
			"model":   a.modelID,
			"content": []interface{}{},
		})
		if err != nil {
			return err
		}
		if err := emit(ev); err != nil {
			return err
		}
	}

	for _, cand := range chunk.Candidates {
		content := cand.Content

		for _, part := range content.Parts {
			if part.Text != "" {
				if !a.textOpen {
					a.textOpen = true
					ev, err := makeEvent("content_block_start", map[string]interface{}{
						"index": 0,
						"content_block": map[string]interface{}{
							"type": "text",
							"text": part.Text,
						},
					})
					if err != nil {
						return err
					}
					if err := emit(ev); err != nil {
						return err
					}
					continue
				}
				ev, err := makeEvent("content_block_delta", map[string]interface{}{
					"index": 0,
					"delta": map[string]interface{}{
						"type": "text_delta",
						"text": part.Text,
					},
				})
				if err != nil {
					return err
				}
				if err := emit(ev); err != nil {
					return err
				}
			}

			if part.FunctionCall != nil {
				if a.textOpen {
					ev, err := makeEvent("content_block_stop", map[string]interface{}{"index": 0})
					if err != nil {
						return err
					}
					if err := emit(ev); err != nil {
						return err
					}
					a.textOpen = false
				}

				ev, err := makeEvent("content_block_start", map[string]interface{}{
					"index": 1,
					"content_block": map[string]interface{}{
						"type":  "tool_use",
						"id":    fmt.Sprintf("toolu_%s", part.FunctionCall.Name),
						"name":  part.FunctionCall.Name,
						"input": part.FunctionCall.Args,
					},
				})
				if err != nil {
					return err
				}
				if err := emit(ev); err != nil {
					return err
				}
				ev2, err := makeEvent("content_block_stop", map[string]interface{}{"index": 1})
				if err != nil {
					return err
				}
				if err := emit(ev2); err != nil {
					return err
				}
			}
		}

		if cand.FinishReason != "" {
			if a.textOpen {
				ev, err := makeEvent("content_block_stop", map[string]interface{}{"index": 0})
				if err != nil {
					return err
				}
				if err := emit(ev); err != nil {
					return err
				}
				a.textOpen = false
			}
			delta := map[string]interface{}{
				"stop_reason": geminiFinishReasonToAnthropic(cand.FinishReason),
			}
			if chunk.UsageMetadata != nil {
				delta["usage"] = map[string]interface{}{
					"input_tokens":  chunk.UsageMetadata.PromptTokenCount,
					"output_tokens": chunk.UsageMetadata.CandidatesTokenCount,
				}
			}
			ev, err := makeEvent("message_delta", map[string]interface{}{
				"delta": delta,
				"usage": delta["usage"],
			})
			if err != nil {
				return err
			}
			return emit(ev)
		}
	}

	return nil
}

func geminiFinishReasonToAnthropic(reason string) string {
	switch reason {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "SAFETY", "RECITATION":
		return "stop_sequence"
	case "TOOL_CALLS":
		return "tool_use"
	default:
		return "end_turn"
	}
}

// Flush sends message_stop.
func (a *geminiStreamAdapter) Flush(emit func(anthropicmessages.MessageStreamEvent) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		ev, err := makeEvent("message_stop", struct{}{})
		if err != nil {
			return err
		}
		return emit(ev)
	}
	return nil
}
