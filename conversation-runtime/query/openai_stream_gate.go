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
