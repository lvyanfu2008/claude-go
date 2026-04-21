package hookexec

import (
	"encoding/json"
	"strings"
)

// BaseHookInput fields align with coreSchemas BaseHookInputSchema.
type BaseHookInput struct {
	SessionID       string `json:"session_id"`
	TranscriptPath  string `json:"transcript_path"`
	Cwd             string `json:"cwd"`
	PermissionMode  string `json:"permission_mode,omitempty"`
	AgentID         string `json:"agent_id,omitempty"`
	AgentType       string `json:"agent_type,omitempty"`
	HookEventName   string `json:"hook_event_name"`
}

func marshalHookInput(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func trimOrDot(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "."
	}
	return s
}
