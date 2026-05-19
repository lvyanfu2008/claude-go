// Package tool: registry.go provides a global per-tool behavior registry that maps
// tool names to optional interface implementations. Call sites check the registry
// first and fall back to hardcoded switches when no implementation is registered.
package tool

import (
	"encoding/json"
	"sync"

	"goc/types"
)

// Behaviors holds optional per-tool interface implementations. Nil fields mean
// "not implemented" — callers should fall back to hardcoded defaults.
type Behaviors struct {
	ActivityDescriber           ActivityDescriber
	SearchOrReadChecker         SearchOrReadChecker
	SearchTextExtractor         SearchTextExtractor
	ResultTruncationChecker     ResultTruncationChecker
	ToolUseSummarizer           ToolUseSummarizer
	OpenWorldChecker            OpenWorldChecker
	UserInteractionChecker      UserInteractionChecker
	AutoClassifierInputProvider AutoClassifierInputProvider
	PermissionMatcherPreparer   PermissionMatcherPreparer
	BackfillObservableInputProvider BackfillObservableInputProvider
}

var (
	regMu     sync.RWMutex
	behaviors = map[string]*Behaviors{}
)

// RegisterBehaviors registers tool behaviors for the given primary name.
// Aliases are not automatically expanded — register each alias separately if needed.
func RegisterBehaviors(name string, b *Behaviors) {
	regMu.Lock()
	defer regMu.Unlock()
	if b == nil {
		delete(behaviors, name)
		return
	}
	behaviors[name] = b
}

// LookupBehaviors returns the registered behaviors for name (exact match, not alias-aware).
func LookupBehaviors(name string) *Behaviors {
	regMu.RLock()
	defer regMu.RUnlock()
	return behaviors[name]
}

// --- Convenience helpers for call sites ---

// GetActivityDescription returns b.GetActivityDescription(input) if registered, or "".
func GetActivityDescription(name string, input json.RawMessage) string {
	b := LookupBehaviors(name)
	if b == nil || b.ActivityDescriber == nil {
		return ""
	}
	return b.ActivityDescriber.GetActivityDescription(input)
}

// IsSearchOrReadCommand returns b.IsSearchOrReadCommand(input) if registered, or nil.
func IsSearchOrReadCommand(name string, input json.RawMessage) *types.SearchOrReadCollapse {
	b := LookupBehaviors(name)
	if b == nil || b.SearchOrReadChecker == nil {
		return nil
	}
	return b.SearchOrReadChecker.IsSearchOrReadCommand(input)
}

// GetToolUseSummary returns b.GetToolUseSummary(input) if registered, or "".
func GetToolUseSummary(name string, input json.RawMessage) string {
	b := LookupBehaviors(name)
	if b == nil || b.ToolUseSummarizer == nil {
		return ""
	}
	return b.ToolUseSummarizer.GetToolUseSummary(input)
}

// ExtractSearchText returns b.ExtractSearchText(output) if registered, or "".
func ExtractSearchText(name string, output json.RawMessage) string {
	b := LookupBehaviors(name)
	if b == nil || b.SearchTextExtractor == nil {
		return ""
	}
	return b.SearchTextExtractor.ExtractSearchText(output)
}

// IsResultTruncated returns b.IsResultTruncated(output) if registered, or false.
func IsResultTruncated(name string, output json.RawMessage) bool {
	b := LookupBehaviors(name)
	if b == nil || b.ResultTruncationChecker == nil {
		return false
	}
	return b.ResultTruncationChecker.IsResultTruncated(output)
}

// IsOpenWorld returns b.IsOpenWorld(input) if registered, or false.
func IsOpenWorld(name string, input json.RawMessage) bool {
	b := LookupBehaviors(name)
	if b == nil || b.OpenWorldChecker == nil {
		return false
	}
	return b.OpenWorldChecker.IsOpenWorld(input)
}

// RequiresUserInteraction returns b.RequiresUserInteraction() if registered, or false.
func RequiresUserInteraction(name string) bool {
	b := LookupBehaviors(name)
	if b == nil || b.UserInteractionChecker == nil {
		return false
	}
	return b.UserInteractionChecker.RequiresUserInteraction()
}

// ToAutoClassifierInput returns b.ToAutoClassifierInput(input) if registered, or "".
func ToAutoClassifierInput(name string, input json.RawMessage) string {
	b := LookupBehaviors(name)
	if b == nil || b.AutoClassifierInputProvider == nil {
		return ""
	}
	return b.AutoClassifierInputProvider.ToAutoClassifierInput(input)
}

// PreparePermissionMatcher returns a pattern-matching closure if registered, or nil.
func PreparePermissionMatcher(name string, input json.RawMessage) (func(pattern string) bool, error) {
	b := LookupBehaviors(name)
	if b == nil || b.PermissionMatcherPreparer == nil {
		return nil, nil
	}
	return b.PermissionMatcherPreparer.PreparePermissionMatcher(input)
}

// BackfillObservableInput calls b.BackfillObservableInput(input) if registered.
func BackfillObservableInput(name string, input map[string]any) {
	b := LookupBehaviors(name)
	if b == nil || b.BackfillObservableInputProvider == nil {
		return
	}
	b.BackfillObservableInputProvider.BackfillObservableInput(input)
}
