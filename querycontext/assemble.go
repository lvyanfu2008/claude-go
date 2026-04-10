package querycontext

import (
	"sort"
	"strings"
)

// AppendSystemContextParts mirrors src/utils/api.ts appendSystemContext (system blocks + one joined context block).
func AppendSystemContextParts(systemPrompt []string, context map[string]string) []string {
	out := make([]string, 0, len(systemPrompt)+1)
	for _, p := range systemPrompt {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	if len(context) == 0 {
		return out
	}
	lines := formatSystemContextLines(context)
	if lines == "" {
		return out
	}
	return append(out, lines)
}

func formatSystemContextLines(context map[string]string) string {
	// Match typical TS object key order (gitStatus before cacheBreaker), then any extras sorted.
	ordered := []string{"gitStatus", "cacheBreaker"}
	var parts []string
	seen := map[string]struct{}{}
	for _, k := range ordered {
		if v, ok := context[k]; ok && strings.TrimSpace(v) != "" {
			parts = append(parts, k+": "+v)
			seen[k] = struct{}{}
		}
	}
	var rest []string
	for k := range context {
		if _, ok := seen[k]; ok {
			continue
		}
		v := context[k]
		if strings.TrimSpace(v) == "" {
			continue
		}
		rest = append(rest, k+": "+v)
	}
	sort.Strings(rest)
	parts = append(parts, rest...)
	return strings.Join(parts, "\n")
}

// FormatUserContextReminder mirrors prependUserContext meta text in src/utils/api.ts (without createUserMessage wrapper).
func FormatUserContextReminder(context map[string]string) string {
	if len(context) == 0 {
		return ""
	}
	// TS getUserContext spreads claudeMd before currentDate; keep that order, then other keys sorted.
	pref := []string{"claudeMd", "currentDate"}
	var blocks []string
	seen := map[string]struct{}{}
	for _, k := range pref {
		if v, ok := context[k]; ok && strings.TrimSpace(v) != "" {
			blocks = append(blocks, "# "+k+"\n"+v)
			seen[k] = struct{}{}
		}
	}
	var restKeys []string
	for k := range context {
		if _, ok := seen[k]; ok {
			continue
		}
		if strings.TrimSpace(context[k]) == "" {
			continue
		}
		restKeys = append(restKeys, k)
	}
	sort.Strings(restKeys)
	for _, k := range restKeys {
		blocks = append(blocks, "# "+k+"\n"+context[k])
	}
	body := strings.Join(blocks, "\n")
	const suffix = "\n\n      IMPORTANT: this context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.\n</system-reminder>\n"
	return `<system-reminder>
As you answer the user's questions, you can use the following context:
` + body + suffix
}
