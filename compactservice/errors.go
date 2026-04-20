package compactservice

import "errors"

// Sentinel error messages mirror the exported TS strings so UI/tests can match them verbatim.
var (
	ErrNotEnoughMessages        = errors.New("Not enough messages to compact.")
	ErrPromptTooLongMessage     = errors.New("Conversation too long. Press esc twice to go up a few messages and try again.")
	ErrUserAbort                = errors.New("API Error: Request was aborted.")
	ErrIncompleteResponse       = errors.New("Compaction interrupted · This may be due to network issues — please try again.")
)

// StartsWithApiErrorPrefix mirrors startsWithApiErrorPrefix in services/api/errors.ts.
// TS treats any text starting with "API Error" as a failure marker.
func StartsWithApiErrorPrefix(s string) bool {
	const prefix = "API Error"
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
