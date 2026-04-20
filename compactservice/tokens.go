package compactservice

import (
	"encoding/json"
	"math"

	"goc/types"
)

// TokenUsage mirrors Anthropic.Usage fields as seen on assistant message.usage in TS.
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens"`
}

// GetTokenUsage mirrors getTokenUsage in utils/tokens.ts. Only non-synthetic
// assistant messages with message.usage present return a non-nil result.
func GetTokenUsage(m types.Message) *TokenUsage {
	if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
		return nil
	}
	if isSyntheticAssistant(m) {
		return nil
	}
	var probe struct {
		Usage *TokenUsage `json:"usage"`
	}
	if err := json.Unmarshal(m.Message, &probe); err != nil {
		return nil
	}
	return probe.Usage
}

// isSyntheticAssistant mirrors the is-synthetic guard around getTokenUsage in TS.
// claude.ts tags API error / withheld assistants with model="synthetic".
func isSyntheticAssistant(m types.Message) bool {
	if len(m.Message) == 0 {
		return false
	}
	var probe struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(m.Message, &probe); err != nil {
		return false
	}
	return probe.Model == "synthetic"
}

// GetTokenCountFromUsage mirrors getTokenCountFromUsage.
func GetTokenCountFromUsage(u TokenUsage) int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens
}

// TokenCountFromLastAPIResponse mirrors tokenCountFromLastAPIResponse.
func TokenCountFromLastAPIResponse(messages []types.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if u := GetTokenUsage(messages[i]); u != nil {
			return GetTokenCountFromUsage(*u)
		}
	}
	return 0
}

// TokenCountWithEstimation mirrors tokenCountWithEstimation in utils/tokens.ts.
// Walks back to the last usage-bearing assistant, rewinds across interleaved siblings with the
// same message.id (parallel-tool-call splitting), then adds rough estimates for any newer messages.
func TokenCountWithEstimation(messages []types.Message) int {
	i := len(messages) - 1
	for i >= 0 {
		m := messages[i]
		u := GetTokenUsage(m)
		if u != nil {
			responseID := assistantInnerMessageID(m)
			if responseID != "" {
				j := i - 1
				for j >= 0 {
					priorID := assistantInnerMessageID(messages[j])
					if priorID == responseID {
						i = j
					} else if priorID != "" {
						break
					}
					j--
				}
			}
			return GetTokenCountFromUsage(*u) + RoughTokenCountEstimationForMessages(messages[i+1:])
		}
		i--
	}
	return RoughTokenCountEstimationForMessages(messages)
}

// RoughTokenCountEstimation mirrors roughTokenCountEstimation(content, bytesPerToken=4).
func RoughTokenCountEstimation(content string) int {
	return int(math.Round(float64(len(content)) / 4.0))
}

// RoughTokenCountEstimationForMessages mirrors roughTokenCountEstimationForMessages in TS.
// Sums text-like content across user/assistant/attachment messages; media blocks use the
// TS constant of 2000 tokens per image/document (see tokenEstimation.ts).
func RoughTokenCountEstimationForMessages(messages []types.Message) int {
	total := 0
	for _, m := range messages {
		total += roughTokenCountEstimationForMessage(m)
	}
	return total
}

func roughTokenCountEstimationForMessage(m types.Message) int {
	switch m.Type {
	case types.MessageTypeAssistant, types.MessageTypeUser:
		if len(m.Message) == 0 {
			return 0
		}
		var probe struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &probe); err != nil {
			return 0
		}
		return roughTokenCountEstimationForContent(probe.Content)
	case types.MessageTypeAttachment:
		// normalizeAttachmentForAPI isn't ported; use attachment JSON as a conservative estimate.
		if len(m.Attachment) == 0 {
			return 0
		}
		return RoughTokenCountEstimation(string(m.Attachment))
	}
	return 0
}

func roughTokenCountEstimationForContent(content json.RawMessage) int {
	if len(content) == 0 {
		return 0
	}
	// content can be a string or an array of content blocks.
	var asString string
	if err := json.Unmarshal(content, &asString); err == nil {
		return RoughTokenCountEstimation(asString)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(content, &arr); err != nil {
		return RoughTokenCountEstimation(string(content))
	}
	total := 0
	for _, b := range arr {
		total += roughTokenCountEstimationForBlock(b)
	}
	return total
}

func roughTokenCountEstimationForBlock(block json.RawMessage) int {
	var s string
	if err := json.Unmarshal(block, &s); err == nil {
		return RoughTokenCountEstimation(s)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(block, &probe); err != nil {
		return RoughTokenCountEstimation(string(block))
	}
	var t string
	if tr, ok := probe["type"]; ok {
		_ = json.Unmarshal(tr, &t)
	}
	switch t {
	case "text":
		var text string
		_ = json.Unmarshal(probe["text"], &text)
		return RoughTokenCountEstimation(text)
	case "image", "document":
		return 2000
	case "tool_result":
		return roughTokenCountEstimationForContent(probe["content"])
	case "tool_use":
		var name string
		_ = json.Unmarshal(probe["name"], &name)
		inputStr := string(probe["input"])
		return RoughTokenCountEstimation(name + inputStr)
	case "thinking":
		var thinking string
		_ = json.Unmarshal(probe["thinking"], &thinking)
		return RoughTokenCountEstimation(thinking)
	case "redacted_thinking":
		var data string
		_ = json.Unmarshal(probe["data"], &data)
		return RoughTokenCountEstimation(data)
	}
	return RoughTokenCountEstimation(string(block))
}
