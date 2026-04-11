package query

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
