package query

import (
	"os"
	"strings"

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

// IsOpenAIThinkingEnabled mirrors src/services/api/openai/index.ts isOpenAIThinkingEnabled.
// DeepSeek-V4-Pro (and API ids containing deepseek-v4-pro) request chain-of-thought params;
// DeepSeek-V4-Flash is treated as a fast non-thinking path unless OPENAI_ENABLE_THINKING=1.
func IsOpenAIThinkingEnabled(model string) bool {
	if isOpenAIEnableThinkingEnvFalsy() {
		return false
	}
	if utils.IsEnvTruthy("OPENAI_ENABLE_THINKING") {
		return true
	}
	m := strings.ToLower(model)
	if isDeepSeekV4FlashModel(model) {
		return false
	}
	return strings.Contains(m, "deepseek-reasoner") ||
		strings.Contains(m, "deepseek-v3.2") ||
		strings.Contains(m, "deepseek-v4-pro")
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
