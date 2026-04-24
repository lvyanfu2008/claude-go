package compactservice

import (
	"errors"
	"strings"
)

// Sentinel error messages mirror the exported TS strings so UI/tests can match them verbatim.
var (
	ErrNotEnoughMessages        = errors.New("Not enough messages to compact.")
	ErrPromptTooLongMessage     = errors.New("Conversation too long. Press esc twice to go up a few messages and try again.")
	ErrUserAbort                = errors.New("API Error: Request was aborted.")
	ErrIncompleteResponse       = errors.New("Compaction interrupted · This may be due to network issues — please try again.")
)

// rateLimitErrorPrefixes mirrors RATE_LIMIT_ERROR_PREFIXES in services/rateLimitMessages.ts.
var rateLimitErrorPrefixes = []string{
	"You've hit your",
	"You've used",
	"You're now using extra usage",
	"You're close to",
	"You're out of extra usage",
}

// IsRateLimitErrorMessage mirrors isRateLimitErrorMessage in services/rateLimitMessages.ts.
func IsRateLimitErrorMessage(text string) bool {
	for _, p := range rateLimitErrorPrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

// StartsWithApiErrorPrefix mirrors startsWithApiErrorPrefix in services/api/errors.ts.
func StartsWithApiErrorPrefix(s string) bool {
	const prefix = "API Error"
	const loginPrefix = "Please run /login · API Error"
	return strings.HasPrefix(s, prefix) || strings.HasPrefix(s, loginPrefix)
}
