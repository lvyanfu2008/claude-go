package query

import "strings"

// systemPromptDynamicBoundary matches prompts.ts SYSTEM_PROMPT_DYNAMIC_BOUNDARY and [commands.SystemPromptDynamicBoundary];
// TS never sends this token on the wire (api.ts splitSysPromptPrefix / assembly skips it).
const systemPromptDynamicBoundary = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__"

// SystemPrompt mirrors src/utils/systemPromptType.ts SystemPrompt (branded string[] in TS).
type SystemPrompt []string

// AsSystemPrompt wraps s as SystemPrompt (TS asSystemPrompt).
func AsSystemPrompt(s []string) SystemPrompt {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return SystemPrompt(out)
}

// StripSystemPromptDynamicBoundaryForAPI removes the internal cache-scope boundary marker from
// the system prompt before HTTP (streaming parity, OpenAI parity, or CallModel). Hosts may still
// build prompts with the marker embedded in one string or as its own slice element.
func StripSystemPromptDynamicBoundaryForAPI(sp SystemPrompt) SystemPrompt {
	if len(sp) == 0 {
		return sp
	}
	joined := strings.Join([]string(sp), "\n\n")
	segs := strings.Split(joined, systemPromptDynamicBoundary)
	var kept []string
	for _, s := range segs {
		if t := strings.TrimSpace(s); t != "" {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return AsSystemPrompt([]string{strings.Join(kept, "\n\n")})
}
