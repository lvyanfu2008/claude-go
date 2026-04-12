package anthropicmessages

import "strings"

// MessagesAPIURL builds the full URL for POST Messages (streaming or not).
//
// Accepts ANTHROPIC_BASE_URL as host-only (e.g. https://api.anthropic.com) or
// already ending in …/v1. If the base already ends with /v1, only "/messages"
// is appended so https://api.anthropic.com/v1 does not become …/v1/v1/messages (404).
func MessagesAPIURL(base string) string {
	s := strings.TrimSpace(base)
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "/v1/messages") {
		return s
	}
	if strings.HasSuffix(s, "/v1") {
		return s + "/messages"
	}
	return s + "/v1/messages"
}
