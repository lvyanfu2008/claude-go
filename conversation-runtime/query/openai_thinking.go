package query

import (
	"os"
	"strings"

	"goc/modelregistry"
	"goc/utils"
)

// isOpenAIEnableThinkingEnvFalsy mirrors src/utils/envUtils.ts isEnvDefinedFalsy(OPENAI_ENABLE_THINKING).
func isOpenAIEnableThinkingEnvFalsy() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("OPENAI_ENABLE_THINKING")))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// isDeepSeekV4FlashModel matches src/services/api/openai/index.ts buildOpenAIRequestBody (v4-flash only).
func isDeepSeekV4FlashModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "deepseek-v4-flash") || strings.Contains(m, "v4-flash")
}

// IsOpenAIThinkingEnabled mirrors claude-code src/api-client/openai/openaiThinking.ts.
// Uses the model family registry for capability detection.
func IsOpenAIThinkingEnabled(model string) bool {
	if isOpenAIEnableThinkingEnvFalsy() {
		return false
	}
	if utils.IsEnvTruthy("OPENAI_ENABLE_THINKING") {
		return true
	}
	caps, ok := modelregistry.Lookup(model)
	if !ok || !caps.SupportsThinking {
		return false
	}
	return caps.DefaultThinkingEnabled
}

// mergeOpenAIThinkingBodyFields injects DeepSeek-style thinking flags into the chat.completions JSON body
// (official API + self-hosted shapes), matching buildOpenAIRequestBody.
func mergeOpenAIThinkingBodyFields(req map[string]any, model string) {
	if isDeepSeekV4FlashModel(model) && !IsOpenAIThinkingEnabled(model) {
		req["thinking"] = map[string]any{"type": "disabled"}
		return
	}
	if !IsOpenAIThinkingEnabled(model) {
		return
	}
	req["thinking"] = map[string]any{"type": "enabled"}
	req["enable_thinking"] = true
	req["chat_template_kwargs"] = map[string]any{"thinking": true}
}
