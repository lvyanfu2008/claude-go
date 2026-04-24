package compactservice

import "testing"

func TestStartsWithApiErrorPrefix(t *testing.T) {
	if !StartsWithApiErrorPrefix("API Error: boom") {
		t.Fatal("expected API Error prefix")
	}
	if !StartsWithApiErrorPrefix("Please run /login · API Error: oauth") {
		t.Fatal("expected login-prefixed API Error")
	}
	if StartsWithApiErrorPrefix("## doc\n\nNeed an API key for streaming.") {
		t.Fatal("must not match API key mention mid-document")
	}
}

func TestIsRateLimitErrorMessage(t *testing.T) {
	if !IsRateLimitErrorMessage("You've hit your rate limit for the day") {
		t.Fatal("expected You've hit your prefix")
	}
	if IsRateLimitErrorMessage("We discussed rate limits in the architecture doc") {
		t.Fatal("must not use substring match")
	}
}
