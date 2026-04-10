package slashresolve

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"goc/types"
)

// ONE_TOKEN_WORDS mirrors src/skills/bundled/loremIpsum.ts (single-token list).
var oneTokenWords = []string{
	"the", "a", "an", "I", "you", "he", "she", "it", "we", "they", "me", "him", "her", "us", "them",
	"my", "your", "his", "its", "our", "this", "that", "what", "who",
	"is", "are", "was", "were", "be", "been", "have", "has", "had", "do", "does", "did",
	"will", "would", "can", "could", "may", "might", "must", "shall", "should",
	"make", "made", "get", "got", "go", "went", "come", "came", "see", "saw", "know", "take", "think",
	"look", "want", "use", "find", "give", "tell", "work", "call", "try", "ask", "need", "feel", "seem", "leave", "put",
	"time", "year", "day", "way", "man", "thing", "life", "hand", "part", "place", "case", "point", "fact",
	"good", "new", "first", "last", "long", "great", "little", "own", "other", "old", "right", "big", "high", "small", "large",
	"next", "early", "young", "few", "public", "bad", "same", "able",
	"in", "on", "at", "to", "for", "of", "with", "from", "by", "about", "like", "through", "over", "before", "between", "under", "since", "without",
	"and", "or", "but", "if", "than", "because", "as", "until", "while", "so", "though", "both", "each", "when", "where", "why", "how",
	"not", "now", "just", "more", "also", "here", "there", "then", "only", "very", "well", "back", "still", "even", "much", "too", "such",
	"never", "again", "most", "once", "off", "away", "down", "out", "up",
	"test", "code", "data", "file", "line", "text", "word", "number", "system", "program", "set", "run", "value", "name", "type", "state", "end", "start",
}

func generateLoremIpsum(targetTokens int) string {
	if targetTokens <= 0 {
		return ""
	}
	var b strings.Builder
	tokens := 0
	for tokens < targetTokens {
		sentenceLen := 10 + rand.IntN(11)
		for i := 0; i < sentenceLen && tokens < targetTokens; i++ {
			w := oneTokenWords[rand.IntN(len(oneTokenWords))]
			b.WriteString(w)
			tokens++
			if i == sentenceLen-1 || tokens >= targetTokens {
				b.WriteString(". ")
			} else {
				b.WriteByte(' ')
			}
		}
		if tokens < targetTokens && rand.Float64() < 0.2 {
			b.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func resolveLoremIpsum(args string) (types.SlashResolveResult, error) {
	args = strings.TrimSpace(args)
	if args != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil || parsed <= 0 {
			return types.SlashResolveResult{
				UserText: "Invalid token count. Please provide a positive number (e.g., /lorem-ipsum 10000).",
				Source:   types.SlashResolveBundledEmbed,
			}, nil
		}
		target := parsed
		const maxTok = 500_000
		if target > maxTok {
			text := fmt.Sprintf("Requested %d tokens, but capped at 500,000 for safety.\n\n%s", target, generateLoremIpsum(maxTok))
			return types.SlashResolveResult{UserText: text, Source: types.SlashResolveBundledEmbed}, nil
		}
		return types.SlashResolveResult{UserText: generateLoremIpsum(target), Source: types.SlashResolveBundledEmbed}, nil
	}
	return types.SlashResolveResult{UserText: generateLoremIpsum(10_000), Source: types.SlashResolveBundledEmbed}, nil
}
