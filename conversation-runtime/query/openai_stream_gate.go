package query

// StreamingUsesOpenAIChat is true when the streaming parity loop should use OpenAI Chat Completions
// (TS queryModelOpenAI path). GOU_QUERY_STREAMING_FORCE_ANTHROPIC=1 disables OpenAI for tests or hosts
// that inject Anthropic StreamPost only.
func StreamingUsesOpenAIChat() bool {
	if envTruthy("GOU_QUERY_STREAMING_FORCE_ANTHROPIC") {
		return false
	}
	return UseOpenAIChatProvider()
}

// OpenAIChatNoStreamEnabled is true when GOU_QUERY_OPENAI_CHAT_NO_STREAM is truthy: OpenAI-compatible
// parity uses one POST /v1/chat/completions (stream:false) per round instead of SSE streaming.
func OpenAIChatNoStreamEnabled() bool {
	return envTruthy("GOU_QUERY_OPENAI_CHAT_NO_STREAM")
}
