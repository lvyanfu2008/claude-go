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
