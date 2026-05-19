// Package tool provides headless helpers and interfaces derived from src/Tool.ts.
// Full Tool<> (Zod, React render*, Ink) remains in TypeScript; Go carries ToolSpec + execution hooks for bridges (ccb-engine, conversation-runtime process-user-input).
package tool

import (
	"context"
	"encoding/json"

	"goc/types"
)

// ToolMatchesName reports whether name equals the tool primary name or any alias (src/Tool.ts toolMatchesName).
func ToolMatchesName(primary string, aliases []string, name string) bool {
	if primary == name {
		return true
	}
	for _, a := range aliases {
		if a == name {
			return true
		}
	}
	return false
}

// FindToolSpecByName returns the first matching spec or nil (src/Tool.ts findToolByName).
func FindToolSpecByName(specs []types.ToolSpec, name string) *types.ToolSpec {
	for i := range specs {
		if ToolMatchesName(specs[i].Name, specs[i].Aliases, name) {
			return &specs[i]
		}
	}
	return nil
}

// FilterToolProgressMessages keeps only tool progress lines, dropping hook_progress (src/Tool.ts filterToolProgressMessages).
func FilterToolProgressMessages(progressMessagesForMessage []types.Message) []types.Message {
	out := make([]types.Message, 0, len(progressMessagesForMessage))
	for _, msg := range progressMessagesForMessage {
		if msg.Type != types.MessageTypeProgress {
			continue
		}
		var probe struct {
			Type string `json:"type"`
		}
		if len(msg.Data) > 0 && json.Unmarshal(msg.Data, &probe) == nil && probe.Type == "hook_progress" {
			continue
		}
		out = append(out, msg)
	}
	return out
}

// CanUseToolFn is the Go stand-in for permission gating before a tool runs (src/hooks/useCanUseTool.ts).
// Return nil to allow; non-nil blocks with an error reason.
type CanUseToolFn func(toolName string, input json.RawMessage, tcx *types.ToolUseContext) error

// Headless is the executable subset of Tool without UI (src/Tool.ts: call, validateInput?, checkPermissions, description).
type Headless interface {
	Spec() types.ToolSpec
	Call(
		ctx context.Context,
		input json.RawMessage,
		tcx *types.ToolUseContext,
		canUse CanUseToolFn,
		parentAssistant json.RawMessage,
		onProgress func(toolUseID string, data json.RawMessage),
	) (*types.ToolRunResult, error)
}

// Describer returns a human-facing tool description string (src/Tool.ts description()).
type Describer interface {
	Description(
		input json.RawMessage,
		isNonInteractive bool,
		toolPerm *types.ToolPermissionContextData,
		toolsJSON json.RawMessage,
	) (string, error)
}

// InputValidator optional (src/Tool.ts validateInput?).
type InputValidator interface {
	ValidateInput(input json.RawMessage, tcx *types.ToolUseContext) (types.ValidationResult, error)
}

// PermissionChecker optional generalization of checkPermissions (src/Tool.ts).
type PermissionChecker interface {
	CheckPermissions(input json.RawMessage, tcx *types.ToolUseContext) (json.RawMessage, error)
}

// PathProvider optional (src/Tool.ts getPath?).
type PathProvider interface {
	GetPath(input json.RawMessage) string
}

// SearchOrReadChecker mirrors Tool.isSearchOrReadCommand? (src/Tool.ts).
// Returns whether this tool use is a search/read/list operation for condensed UI display.
type SearchOrReadChecker interface {
	IsSearchOrReadCommand(input json.RawMessage) *types.SearchOrReadCollapse
}

// SearchTextExtractor mirrors Tool.extractSearchText? (src/Tool.ts).
// Returns the flattened text content of the tool result for transcript search indexing.
type SearchTextExtractor interface {
	ExtractSearchText(output json.RawMessage) string
}

// ResultTruncationChecker mirrors Tool.isResultTruncated? (src/Tool.ts).
// Returns true when the non-verbose rendering of this output is truncated.
type ResultTruncationChecker interface {
	IsResultTruncated(output json.RawMessage) bool
}

// ActivityDescriber mirrors Tool.getActivityDescription? (src/Tool.ts).
// Returns a human-readable present-tense activity description (e.g. "Reading src/foo.ts").
// Returns "" to fall back to the tool name.
type ActivityDescriber interface {
	GetActivityDescription(input json.RawMessage) string
}

// ToolUseSummarizer mirrors Tool.getToolUseSummary? (src/Tool.ts).
// Returns a short summary of this tool use for compact views, or "" for none.
type ToolUseSummarizer interface {
	GetToolUseSummary(input json.RawMessage) string
}

// OpenWorldChecker mirrors Tool.isOpenWorld? (src/Tool.ts).
type OpenWorldChecker interface {
	IsOpenWorld(input json.RawMessage) bool
}

// UserInteractionChecker mirrors Tool.requiresUserInteraction? (src/Tool.ts).
type UserInteractionChecker interface {
	RequiresUserInteraction() bool
}

// AutoClassifierInputProvider mirrors Tool.toAutoClassifierInput (src/Tool.ts).
// Returns a compact representation for the auto-mode security classifier.
// Returns "" to skip this tool in the classifier transcript.
type AutoClassifierInputProvider interface {
	ToAutoClassifierInput(input json.RawMessage) string
}

// PermissionMatcherPreparer mirrors Tool.preparePermissionMatcher? (src/Tool.ts).
// Returns a closure that tests whether a hook permission-rule pattern matches this tool input.
type PermissionMatcherPreparer interface {
	PreparePermissionMatcher(input json.RawMessage) (func(pattern string) bool, error)
}

// BackfillObservableInputProvider mirrors Tool.backfillObservableInput? (src/Tool.ts).
// Mutates input in-place to add legacy/derived fields before observers see it.
type BackfillObservableInputProvider interface {
	BackfillObservableInput(input map[string]any)
}
