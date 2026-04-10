package appstate

import "goc/types"

// AllowedPrompt mirrors src/tools/ExitPlanModeTool/ExitPlanModeV2Tool.ts AllowedPrompt (zod schema).
type AllowedPrompt struct {
	Tool   string `json:"tool"` // TS: enum e.g. "Bash"
	Prompt string `json:"prompt"`
}

// InitialMessage mirrors src/state/AppStateStore.ts AppState.initialMessage (non-null branch).
// TS: null when unset; in Go use nil *InitialMessage (JSON null).
type InitialMessage struct {
	Message        types.Message         `json:"message"` // TS UserMessage
	ClearContext   *bool                 `json:"clearContext,omitempty"`
	Mode           *types.PermissionMode `json:"mode,omitempty"`
	AllowedPrompts []AllowedPrompt       `json:"allowedPrompts,omitempty"`
}
