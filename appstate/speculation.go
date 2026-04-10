package appstate

import (
	"encoding/json"

	"goc/types"
)

// Completion boundary kinds (src/state/AppStateStore.ts CompletionBoundary).
const (
	BoundaryComplete   = "complete"
	BoundaryBash       = "bash"
	BoundaryEdit       = "edit"
	BoundaryDeniedTool = "denied_tool"
)

// CompletionBoundary mirrors src/state/AppStateStore.ts CompletionBoundary (discriminated by Type).
type CompletionBoundary struct {
	Type         string `json:"type"`
	CompletedAt  int64  `json:"completedAt"`
	OutputTokens int    `json:"outputTokens,omitempty"` // type complete
	Command      string `json:"command,omitempty"`      // type bash
	ToolName     string `json:"toolName,omitempty"`     // type edit | denied_tool
	FilePath     string `json:"filePath,omitempty"`     // type edit
	Detail       string `json:"detail,omitempty"`       // type denied_tool
}

// SpeculationResult mirrors src/state/AppStateStore.ts SpeculationResult.
type SpeculationResult struct {
	Messages    []types.Message     `json:"messages"`
	Boundary    *CompletionBoundary `json:"boundary"`
	TimeSavedMs int64               `json:"timeSavedMs"`
}

const (
	SpeculationStatusIdle   = "idle"
	SpeculationStatusActive = "active"
)

// PipelinedSuggestion mirrors TS SpeculationState active.pipelinedSuggestion (data only).
type PipelinedSuggestion struct {
	Text                string  `json:"text"`
	PromptID            string  `json:"promptId"` // user_intent | stated_intent
	GenerationRequestID *string `json:"generationRequestId"`
}

// SpeculationState mirrors the JSON-serializable part of src/state/AppStateStore.ts SpeculationState.
// TS active branch also has abort, messagesRef, writtenPathsRef, contextRef (refs/functions) — omitted here.
type SpeculationState struct {
	Status string `json:"status"`

	// Active-only (omitted when idle).
	ID                  string               `json:"id,omitempty"`
	StartTime           int64                `json:"startTime,omitempty"`
	Messages            []types.Message      `json:"messages,omitempty"`
	WrittenPaths        []string             `json:"writtenPaths,omitempty"`
	Boundary            *CompletionBoundary  `json:"boundary,omitempty"`
	SuggestionLength    int                  `json:"suggestionLength,omitempty"`
	ToolUseCount        int                  `json:"toolUseCount,omitempty"`
	IsPipelined         bool                 `json:"isPipelined,omitempty"`
	Context             json.RawMessage      `json:"context,omitempty"` // REPLHookContext snapshot
	PipelinedSuggestion *PipelinedSuggestion `json:"pipelinedSuggestion,omitempty"`
}

// IdleSpeculationState matches TS IDLE_SPECULATION_STATE / { status: 'idle' }.
func IdleSpeculationState() SpeculationState {
	return SpeculationState{Status: SpeculationStatusIdle}
}

func normalizeSpeculationCollections(s *SpeculationState) {
	if s == nil || s.Status != SpeculationStatusActive {
		return
	}
	if s.Messages == nil {
		s.Messages = []types.Message{}
	}
	if s.WrittenPaths == nil {
		s.WrittenPaths = []string{}
	}
}
