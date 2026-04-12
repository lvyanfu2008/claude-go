package anthropicmessages

import "testing"

func TestMessagesAPIURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		base, want string
	}{
		{"https://api.anthropic.com", "https://api.anthropic.com/v1/messages"},
		{"https://api.anthropic.com/", "https://api.anthropic.com/v1/messages"},
		{"https://api.anthropic.com/v1", "https://api.anthropic.com/v1/messages"},
		{"https://api.anthropic.com/v1/", "https://api.anthropic.com/v1/messages"},
		{"https://api.anthropic.com/v1/messages", "https://api.anthropic.com/v1/messages"},
	}
	for _, tc := range cases {
		if got := MessagesAPIURL(tc.base); got != tc.want {
			t.Fatalf("MessagesAPIURL(%q) = %q want %q", tc.base, got, tc.want)
		}
	}
}
