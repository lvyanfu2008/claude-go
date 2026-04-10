// Mirrors src/utils/hooks.ts HookBlockingError and AggregatedHookResult (UserPromptSubmit path).
package types

import "encoding/json"

// HookBlockingError mirrors src/utils/hooks.ts HookBlockingError.
type HookBlockingError struct {
	BlockingError string `json:"blockingError"`
	Command       string `json:"command"`
}

// AggregatedHookResult mirrors src/utils/hooks.ts AggregatedHookResult.
// Fields that reference MCP SDK or complex unions use json.RawMessage where needed.
type AggregatedHookResult struct {
	Message json.RawMessage `json:"message,omitempty"` // HookResultMessage

	BlockingError *HookBlockingError `json:"blockingError,omitempty"`

	PreventContinuation *bool `json:"preventContinuation,omitempty"`
	StopReason          *string `json:"stopReason,omitempty"`

	HookPermissionDecisionReason *string `json:"hookPermissionDecisionReason,omitempty"`
	HookSource                   *string `json:"hookSource,omitempty"`

	PermissionBehavior *string `json:"permissionBehavior,omitempty"` // PermissionResult behavior union — simplified

	AdditionalContexts   []string       `json:"additionalContexts,omitempty"`
	InitialUserMessage   *string        `json:"initialUserMessage,omitempty"`
	UpdatedInput         json.RawMessage `json:"updatedInput,omitempty"`
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitempty"`
	PermissionRequestResult json.RawMessage `json:"permissionRequestResult,omitempty"`

	WatchPaths []string `json:"watchPaths,omitempty"`

	ElicitationResponse        json.RawMessage `json:"elicitationResponse,omitempty"`
	ElicitationResultResponse json.RawMessage `json:"elicitationResultResponse,omitempty"`

	Retry *bool `json:"retry,omitempty"`
}
