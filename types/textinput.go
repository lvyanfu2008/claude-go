// Mirrors src/types/textInputTypes.ts PromptInputMode and related input modes.
package types

// PromptInputMode is the REPL input strip mode (src/types/textInputTypes.ts).
type PromptInputMode string

const (
	PromptInputModeBash                  PromptInputMode = "bash"
	PromptInputModePrompt                PromptInputMode = "prompt"
	PromptInputModeOrphanedPermission    PromptInputMode = "orphaned-permission"
	PromptInputModeTaskNotification      PromptInputMode = "task-notification"
)
