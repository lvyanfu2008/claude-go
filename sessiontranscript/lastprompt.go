package sessiontranscript

import (
	"encoding/json"
	"regexp"
	"strings"

	"goc/commands"
	"goc/types"
)

// Mirrors src/constants/xml.ts COMMAND_NAME_TAG.
const commandNameTag = "command-name"

// Same as sessionStorage.ts SKIP_FIRST_PROMPT_PATTERN (kept in sync with sessionStoragePortable.ts).
var skipFirstPromptPattern = regexp.MustCompile(`^(?:\s*<[a-z][\w-]*[\s>]|\[Request interrupted by user[^\]]*\])`)

// extractTag mirrors src/utils/extractTag.ts (first match at nesting depth 0 for tagName).
func extractTag(html, tagName string) string {
	html = strings.TrimSpace(html)
	tagName = strings.TrimSpace(tagName)
	if html == "" || tagName == "" {
		return ""
	}
	escaped := regexp.QuoteMeta(tagName)
	openingTag := regexp.MustCompile(`(?i)<` + escaped + `(?:\s+[^>]*?)?>`)
	closingTag := regexp.MustCompile(`(?i)</` + escaped + `>`)
	pattern := regexp.MustCompile(`(?is)<` + escaped + `(?:\s+[^>]*)?>([\s\S]*?)</` + escaped + `>`)

	lastIndex := 0
	for {
		rest := html[lastIndex:]
		loc := pattern.FindStringSubmatchIndex(rest)
		if loc == nil {
			return ""
		}
		globalStart := lastIndex + loc[0]
		globalEnd := lastIndex + loc[1]
		contentStart := lastIndex + loc[2]
		contentEnd := lastIndex + loc[3]
		content := html[contentStart:contentEnd]
		before := html[lastIndex:globalStart]

		depth := len(openingTag.FindAllStringIndex(before, -1))
		depth -= len(closingTag.FindAllStringIndex(before, -1))
		if depth == 0 && content != "" {
			return content
		}
		lastIndex = globalEnd
	}
}

func userMessageTextBlocks(msg types.Message) []string {
	if len(msg.Message) == 0 {
		return nil
	}
	var wrap struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &wrap); err != nil || len(wrap.Content) == 0 {
		return nil
	}
	var asStr string
	if err := json.Unmarshal(wrap.Content, &asStr); err == nil {
		if strings.TrimSpace(asStr) == "" {
			return nil
		}
		return []string{asStr}
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(wrap.Content, &blocks); err != nil {
		return nil
	}
	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			texts = append(texts, b.Text)
		}
	}
	return texts
}

// FirstMeaningfulUserMessageTextContent mirrors sessionStorage.ts getFirstMeaningfulUserMessageTextContent.
func FirstMeaningfulUserMessageTextContent(transcript []types.Message) string {
	builtins := commands.BuiltinCommandNameSet()
	for _, msg := range transcript {
		if msg.Type != types.MessageTypeUser {
			continue
		}
		if msg.IsMeta != nil && *msg.IsMeta {
			continue
		}
		if msg.IsCompactSummary != nil && *msg.IsCompactSummary {
			continue
		}
		for _, textContent := range userMessageTextBlocks(msg) {
			if textContent == "" {
				continue
			}
			if cmdTag := extractTag(textContent, commandNameTag); cmdTag != "" {
				commandName := strings.TrimPrefix(cmdTag, "/")
				if _, ok := builtins[commandName]; ok {
					continue
				}
				args := strings.TrimSpace(extractTag(textContent, "command-args"))
				if args == "" {
					continue
				}
				return strings.TrimSpace(cmdTag + " " + args)
			}
			if bashInput := extractTag(textContent, "bash-input"); bashInput != "" {
				return "! " + bashInput
			}
			if skipFirstPromptPattern.MatchString(textContent) {
				continue
			}
			return textContent
		}
	}
	return ""
}

// FlattenLastPromptCache mirrors sessionStorage insertMessageChain lastPrompt cache (single-line, max 200 + ellipsis).
func FlattenLastPromptCache(text string) string {
	flat := strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	flat = strings.TrimSpace(flat)
	if flat == "" {
		return ""
	}
	const max = 200
	rr := []rune(flat)
	if len(rr) <= max {
		return flat
	}
	// TS: slice(0,200).trim() + '…' (200 code units; use runes for ASCII/BMP parity).
	s := strings.TrimSpace(string(rr[:max]))
	return s + "…"
}
