package builtin

import (
	_ "embed"
	"strings"
)

//go:embed promptdata/verification.md
var verificationPromptMD string

// VerificationSystemPrompt returns the verification agent system prompt (from verificationAgent.ts).
func VerificationSystemPrompt() string {
	return strings.ReplaceAll(strings.ReplaceAll(verificationPromptMD, "Bash", ToolBash), "WebFetch", ToolWebFetch)
}
