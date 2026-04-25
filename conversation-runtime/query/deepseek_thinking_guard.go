package query

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"goc/types"
)

// DeepSeekThinkingRetryUserEN mirrors TS [DEEPSEEK_THINKING_RETRY_USER_EN].
const DeepSeekThinkingRetryUserEN = "Your previous response did not include the required chain-of-thought for this model " +
	"(reasoning / thinking output). In thinking mode you must emit the API's " +
	"reasoning content before the final answer. Please answer again; follow any " +
	"tool instructions if they still apply."

// isClaudeCodeDeepSeekStrictEnvFalsy mirrors src/utils/envUtils.ts isEnvDefinedFalsy for CLAUDE_CODE_DEEPSEEK_STRICT_THINKING.
func isClaudeCodeDeepSeekStrictEnvFalsy() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING")))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// IsDeepSeekStrictThinkingGuardEnabled mirrors TS [isDeepSeekStrictThinkingGuardEnabled]: default on when env unset.
func IsDeepSeekStrictThinkingGuardEnabled() bool {
	return !isClaudeCodeDeepSeekStrictEnvFalsy()
}

// OpenAIEnforcesReasoningInThinkingMode mirrors TS [openAIEnforcesReasoningInThinkingMode].
func OpenAIEnforcesReasoningInThinkingMode(openaiModel string, enableThinking bool) bool {
	if !IsDeepSeekStrictThinkingGuardEnabled() || !enableThinking {
		return false
	}
	m := strings.ToLower(openaiModel)
	if isDeepSeekV4FlashModel(openaiModel) {
		return true
	}
	return strings.Contains(m, "deepseek-v4-pro")
}

// GetDeepSeekStrictThinkingMaxAttempts mirrors TS getDeepSeekStrictThinkingMaxAttempts (default 2, range 1–5).
func GetDeepSeekStrictThinkingMaxAttempts() int {
	raw := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING_ATTEMPTS"))
	if raw == "" {
		return 2
	}
	d, err := strconv.Atoi(raw)
	if err != nil || d < 1 || d > 5 {
		return 2
	}
	return d
}

// openAIEnableThinkingForRequest mirrors TS queryModelOpenAI enableThinking:
// IsOpenAIThinkingEnabled(model) || (transcript has assistant thinking && OPENAI_ENABLE_THINKING not defined-falsy).
func openAIEnableThinkingForRequest(model string, work []types.Message) bool {
	if IsOpenAIThinkingEnabled(model) {
		return true
	}
	if isOpenAIEnableThinkingEnvFalsy() {
		return false
	}
	return workMessagesHaveAssistantThinking(work)
}

func workMessagesHaveAssistantThinking(work []types.Message) bool {
	for _, m := range work {
		if m.Type != types.MessageTypeAssistant {
			continue
		}
		if len(m.Message) == 0 {
			continue
		}
		var inner struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &inner); err != nil {
			continue
		}
		if wireAssistantContentHasThinking(inner.Content) {
			return true
		}
	}
	return false
}

// assistantWireMessageHasNonEmptyThinkingBlock mirrors TS [assistantHasNonEmptyThinkingBlock] on types.Message `message` JSON.
func assistantWireMessageHasNonEmptyThinkingBlock(inner json.RawMessage) bool {
	if len(inner) == 0 || string(inner) == "null" {
		return false
	}
	var wrap struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(inner, &wrap); err != nil {
		return false
	}
	if len(wrap.Content) == 0 {
		return false
	}
	var blocks []map[string]any
	if err := json.Unmarshal(wrap.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		switch typ {
		case "thinking":
			if s, ok := b["thinking"].(string); ok && strings.TrimSpace(s) != "" {
				return true
			}
		case "redacted_thinking":
			return true
		}
	}
	return false
}

// deepseekThinkingErrorAssistantMessage mirrors createAssistantAPIErrorMessage content from TS.
const deepseekThinkingErrorText = "In thinking mode this model must return a non-empty chain-of-thought (reasoning) block, but the response had none. " +
	"Set CLAUDE_CODE_DEEPSEEK_STRICT_THINKING=0 to disable this check."

// buildDeepseekThinkingErrorAssistant yields the same user-visible text as TS.
func buildDeepseekThinkingErrorAssistant(uuid string) (types.Message, error) {
	t := true
	inner := map[string]any{
		"role": "assistant",
		"content": []any{
			map[string]any{
				"type": "text",
				"text": deepseekThinkingErrorText,
			},
		},
	}
	raw, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}, err
	}
	return types.Message{
		Type:              types.MessageTypeAssistant,
		UUID:              uuid,
		Message:           raw,
		IsApiErrorMessage: &t,
	}, nil
}
