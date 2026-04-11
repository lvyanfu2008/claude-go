package query

import (
	"encoding/json"

	"goc/types"
)

// State is mutable cross-iteration state for queryLoop (TS query.ts State).
type State struct {
	Messages                     []types.Message
	ToolUseContext               types.ToolUseContext
	AutoCompactTracking          json.RawMessage `json:"-"` // TS AutoCompactTrackingState | undefined
	MaxOutputTokensRecoveryCount int
	HasAttemptedReactiveCompact  bool
	MaxOutputTokensOverride      *int
	PendingToolUseSummary        any // TS Promise<ToolUseSummaryMessage | null> | undefined
	StopHookActive               *bool
	TurnCount                    int
	Transition                   *Continue
}

// NewStateFromParams seeds State from QueryParams (first iteration).
func NewStateFromParams(p QueryParams) State {
	var act json.RawMessage
	if len(p.AutoCompactTracking) > 0 {
		act = append(json.RawMessage(nil), p.AutoCompactTracking...)
	}
	return State{
		Messages:                     append([]types.Message(nil), p.Messages...),
		ToolUseContext:               p.ToolUseContext,
		AutoCompactTracking:          act,
		MaxOutputTokensOverride:      p.MaxOutputTokensOverride,
		MaxOutputTokensRecoveryCount: 0,
		HasAttemptedReactiveCompact:  false,
		TurnCount:                    1,
	}
}
