package query

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"goc/querycontext"
	"goc/types"
)

// AppendSystemContext mirrors src/utils/api.ts appendSystemContext.
func AppendSystemContext(system SystemPrompt, context map[string]string) SystemPrompt {
	if len(context) == 0 {
		out := make([]string, 0, len(system))
		out = append(out, system...)
		return AsSystemPrompt(out)
	}
	block := querycontext.FormatSystemContextLines(context)
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

// LogQueryUserContextIfEnabled writes [QueryParams.UserContext] as JSON to stderr when
// GOU_QUERY_LOG_USER_CONTEXT is truthy (1/true/yes/on). Large payloads are truncated after 64KiB.
// Use to compare hosts vs TS: same env gate pattern as [BuildQueryConfig] gates.
func LogQueryUserContextIfEnabled(tag string, context map[string]string) {
	if !envTruthy("GOU_QUERY_LOG_USER_CONTEXT") {
		return
	}
	const maxJSON = 64 << 10
	n := 0
	if context != nil {
		n = len(context)
	}
	raw, err := json.Marshal(context)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[query UserContext %s] marshal error: %v\n", tag, err)
		return
	}
	s := string(raw)
	trunc := ""
	if len(s) > maxJSON {
		s = s[:maxJSON]
		trunc = fmt.Sprintf(" [truncated from %d bytes]", len(raw))
	}
	_, _ = fmt.Fprintf(os.Stderr, "[query UserContext %s] keyCount=%d json=%s%s\n", tag, n, s, trunc)
}

// PrependUserContext mirrors src/utils/api.ts prependUserContext (production path).
// TS skips prepending when NODE_ENV === 'test'; use [SkipUserContextInTest] for the same in Go tests.
var SkipUserContextInTest bool

// PrependUserContext prepends a meta user message when context is non-empty (same slice order as TS).
// [messagesapi.NormalizeMessagesForAPI] only merges consecutive user rows (mergeUserMessages); it does not
// move a leading meta user across assistant boundaries onto the trailing user (TS normalize does not either).
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
