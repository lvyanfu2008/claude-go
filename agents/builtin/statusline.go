package builtin

import _ "embed"

//go:embed promptdata/statusline.md
var statuslinePromptMD string

// StatuslineSystemPrompt returns the statusline-setup agent system prompt (from statuslineSetup.ts).
func StatuslineSystemPrompt() string {
	return statuslinePromptMD
}
