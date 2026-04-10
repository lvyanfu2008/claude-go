package processuserinput

import (
	"regexp"
	"strings"
)

// Mirrors src/utils/userPromptKeywords.ts (same patterns).

var negativeKeywordPattern = regexp.MustCompile(
	`\b(wtf|wth|ffs|omfg|shit(ty|tiest)?|dumbass|horrible|awful|piss(ed|ing)? off|piece of (shit|crap|junk)|what the (fuck|hell)|fucking? (broken|useless|terrible|awful|horrible)|fuck you|screw (this|you)|so frustrating|this sucks|damn it)\b`,
)

var keepGoingKeywordPattern = regexp.MustCompile(`\b(keep going|go on)\b`)

// MatchesNegativeKeyword mirrors matchesNegativeKeyword in userPromptKeywords.ts.
func MatchesNegativeKeyword(input string) bool {
	return negativeKeywordPattern.MatchString(strings.ToLower(input))
}

// MatchesKeepGoingKeyword mirrors matchesKeepGoingKeyword in userPromptKeywords.ts.
func MatchesKeepGoingKeyword(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "continue" {
		return true
	}
	return keepGoingKeywordPattern.MatchString(lower)
}
