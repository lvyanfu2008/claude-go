package appstate

import (
	"encoding/json"
	"os"
	"strings"
)

// FileAttributionState mirrors src/types/logs.ts FileAttributionState.
type FileAttributionState struct {
	ContentHash        string `json:"contentHash"`
	ClaudeContribution int    `json:"claudeContribution"`
	Mtime              int64  `json:"mtime"`
}

// SessionBaseline mirrors entries in TS AttributionState.sessionBaselines Map values.
type SessionBaseline struct {
	ContentHash string `json:"contentHash"`
	Mtime       int64  `json:"mtime"`
}

// AttributionState mirrors src/utils/commitAttribution.ts AttributionState (Map fields → maps for JSON).
type AttributionState struct {
	FileStates                        map[string]FileAttributionState `json:"fileStates"`
	SessionBaselines                  map[string]SessionBaseline      `json:"sessionBaselines"`
	Surface                           string                          `json:"surface"`
	StartingHeadSha                   *string                         `json:"startingHeadSha"`
	PromptCount                       int                             `json:"promptCount"`
	PromptCountAtLastCommit           int                             `json:"promptCountAtLastCommit"`
	PermissionPromptCount             int                             `json:"permissionPromptCount"`
	PermissionPromptCountAtLastCommit int                             `json:"permissionPromptCountAtLastCommit"`
	EscapeCount                       int                             `json:"escapeCount"`
	EscapeCountAtLastCommit           int                             `json:"escapeCountAtLastCommit"`
}

func defaultClientSurface() string {
	if s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT")); s != "" {
		return s
	}
	return "cli"
}

// EmptyAttributionState mirrors createEmptyAttributionState() (empty maps, zero counters).
func EmptyAttributionState() AttributionState {
	return AttributionState{
		FileStates:                        make(map[string]FileAttributionState),
		SessionBaselines:                  make(map[string]SessionBaseline),
		Surface:                           defaultClientSurface(),
		StartingHeadSha:                   nil,
		PromptCount:                       0,
		PromptCountAtLastCommit:           0,
		PermissionPromptCount:             0,
		PermissionPromptCountAtLastCommit: 0,
		EscapeCount:                       0,
		EscapeCountAtLastCommit:           0,
	}
}

// MarshalJSON emits {} for empty maps (TS Map JSON is {}), not null.
func (a AttributionState) MarshalJSON() ([]byte, error) {
	type out struct {
		FileStates                        map[string]FileAttributionState `json:"fileStates"`
		SessionBaselines                  map[string]SessionBaseline      `json:"sessionBaselines"`
		Surface                           string                          `json:"surface"`
		StartingHeadSha                   *string                         `json:"startingHeadSha"`
		PromptCount                       int                             `json:"promptCount"`
		PromptCountAtLastCommit           int                             `json:"promptCountAtLastCommit"`
		PermissionPromptCount             int                             `json:"permissionPromptCount"`
		PermissionPromptCountAtLastCommit int                             `json:"permissionPromptCountAtLastCommit"`
		EscapeCount                       int                             `json:"escapeCount"`
		EscapeCountAtLastCommit           int                             `json:"escapeCountAtLastCommit"`
	}
	fs := a.FileStates
	if fs == nil {
		fs = make(map[string]FileAttributionState)
	}
	sb := a.SessionBaselines
	if sb == nil {
		sb = make(map[string]SessionBaseline)
	}
	return json.Marshal(out{
		FileStates: fs, SessionBaselines: sb, Surface: a.Surface, StartingHeadSha: a.StartingHeadSha,
		PromptCount: a.PromptCount, PromptCountAtLastCommit: a.PromptCountAtLastCommit,
		PermissionPromptCount: a.PermissionPromptCount, PermissionPromptCountAtLastCommit: a.PermissionPromptCountAtLastCommit,
		EscapeCount: a.EscapeCount, EscapeCountAtLastCommit: a.EscapeCountAtLastCommit,
	})
}

// UnmarshalJSON normalizes null/omitted maps to empty maps (parity with TS empty Map).
func (a *AttributionState) UnmarshalJSON(data []byte) error {
	var s struct {
		FileStates                        map[string]FileAttributionState `json:"fileStates"`
		SessionBaselines                  map[string]SessionBaseline      `json:"sessionBaselines"`
		Surface                           string                          `json:"surface"`
		StartingHeadSha                   *string                         `json:"startingHeadSha"`
		PromptCount                       int                             `json:"promptCount"`
		PromptCountAtLastCommit           int                             `json:"promptCountAtLastCommit"`
		PermissionPromptCount             int                             `json:"permissionPromptCount"`
		PermissionPromptCountAtLastCommit int                             `json:"permissionPromptCountAtLastCommit"`
		EscapeCount                       int                             `json:"escapeCount"`
		EscapeCountAtLastCommit           int                             `json:"escapeCountAtLastCommit"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*a = AttributionState{
		FileStates:                        s.FileStates,
		SessionBaselines:                  s.SessionBaselines,
		Surface:                           s.Surface,
		StartingHeadSha:                   s.StartingHeadSha,
		PromptCount:                       s.PromptCount,
		PromptCountAtLastCommit:           s.PromptCountAtLastCommit,
		PermissionPromptCount:             s.PermissionPromptCount,
		PermissionPromptCountAtLastCommit: s.PermissionPromptCountAtLastCommit,
		EscapeCount:                       s.EscapeCount,
		EscapeCountAtLastCommit:           s.EscapeCountAtLastCommit,
	}
	if a.FileStates == nil {
		a.FileStates = make(map[string]FileAttributionState)
	}
	if a.SessionBaselines == nil {
		a.SessionBaselines = make(map[string]SessionBaseline)
	}
	return nil
}
