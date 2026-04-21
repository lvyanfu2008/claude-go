package hookexec

import (
	"regexp"
	"strings"
)

// MatchesPattern mirrors src/utils/hooks.ts matchesPattern for hook matcher selection.
// matchQuery is the hook-specific query (e.g. InstructionsLoaded load_reason, SessionStart source).
func MatchesPattern(matchQuery, matcher string) bool {
	matcher = strings.TrimSpace(matcher)
	if matcher == "" || matcher == "*" {
		return true
	}
	mq := strings.TrimSpace(matchQuery)
	// Pipe-separated exact matches / simple token (TS: /^[a-zA-Z0-9_|]+$/)
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_|]+$`, matcher); matched {
		if strings.Contains(matcher, "|") {
			for _, p := range strings.Split(matcher, "|") {
				p = strings.TrimSpace(p)
				if p != "" && mq == p {
					return true
				}
			}
			return false
		}
		return mq == matcher
	}
	re, err := regexp.Compile(matcher)
	if err != nil {
		return false
	}
	return re.MatchString(mq)
}
