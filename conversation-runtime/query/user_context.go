package query

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"goc/types"
)

// AppendSystemContext mirrors src/utils/api.ts appendSystemContext.
func AppendSystemContext(system SystemPrompt, context map[string]string) SystemPrompt {
	if len(context) == 0 {
		out := make([]string, 0, len(system))
		out = append(out, system...)
		return AsSystemPrompt(out)
	}
	keys := make([]string, 0, len(context))
	for k := range context {
		if strings.TrimSpace(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", k, context[k]))
	}
	block := strings.Join(lines, "\n")
	var joined []string
	joined = append(joined, system...)
	if block != "" {
		joined = append(joined, block)
	}
	out := make([]string, 0, len(joined))
	for _, s := range joined {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return AsSystemPrompt(out)
}

// PrependUserContext mirrors src/utils/api.ts prependUserContext (production path).
// TS skips prepending when NODE_ENV === 'test'; use [SkipUserContextInTest] for the same in Go tests.
var SkipUserContextInTest bool

// PrependUserContext prepends a meta user message when context is non-empty.
func PrependUserContext(messages []types.Message, context map[string]string) []types.Message {
	if SkipUserContextInTest {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	if len(context) == 0 {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	keys := make([]string, 0, len(context))
	for k := range context {
		if strings.TrimSpace(k) != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	// Stable order: sort keys for reproducible API bodies (TS uses Object.entries order).
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("<system-reminder>\nAs you answer the user's questions, you can use the following context:\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "# %s\n%s\n", k, context[k])
	}
	b.WriteString("\nIMPORTANT: this context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.\n</system-reminder>\n")

	content := b.String()
	meta := true
	inner, err := json.Marshal(map[string]any{"role": "user", "content": content})
	if err != nil {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	prefix := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    randomUUID(),
		Message: json.RawMessage(inner),
		IsMeta:  &meta,
	}
	out := make([]types.Message, 0, len(messages)+1)
	out = append(out, prefix)
	out = append(out, messages...)
	return out
}
