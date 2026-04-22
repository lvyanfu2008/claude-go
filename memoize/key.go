package memoize

import "encoding/json"

// KeyJSON mirrors src/utils/memoize.ts + jsonStringify: stable string keys for memos.
func KeyJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MustKeyJSON is like [KeyJSON] but panics on marshal error; only for static-shapes / tests.
func MustKeyJSON(v any) string {
	s, err := KeyJSON(v)
	if err != nil {
		panic(err)
	}
	return s
}
