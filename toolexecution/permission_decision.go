package toolexecution

import (
	"context"
	"encoding/json"
)

// PermissionBehavior mirrors PermissionDecision.behavior in src/utils/permissions (subset for Go parity).
type PermissionBehavior string

const (
	PermissionAllow PermissionBehavior = "allow"
	PermissionDeny  PermissionBehavior = "deny"
	PermissionAsk   PermissionBehavior = "ask"
)

// PermissionDecision is the headless subset of TS PermissionDecision / PermissionResult used before tool execution.
type PermissionDecision struct {
	Behavior PermissionBehavior `json:"behavior"`
	Message  string             `json:"message,omitempty"`
}

// AllowDecision returns an allow decision (TS allow).
func AllowDecision() PermissionDecision {
	return PermissionDecision{Behavior: PermissionAllow}
}

// DenyDecision returns deny with a user-visible message.
func DenyDecision(message string) PermissionDecision {
	return PermissionDecision{Behavior: PermissionDeny, Message: message}
}

// AskDecision returns ask (host or [ExecutionDeps.AskResolver] must resolve).
func AskDecision(message string) PermissionDecision {
	return PermissionDecision{Behavior: PermissionAsk, Message: message}
}

// QueryCanUseToolFn mirrors query.ts CanUseToolFn outcome shape for headless Go (tool name + ids + raw JSON input).
type QueryCanUseToolFn func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error)

// LegacyBoolQueryGate adapts a legacy (allowed bool, err) gate to [QueryCanUseToolFn].
func LegacyBoolQueryGate(fn func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (bool, error)) QueryCanUseToolFn {
	if fn == nil {
		return nil
	}
	return func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
		ok, err := fn(ctx, toolName, toolUseID, input)
		if err != nil {
			return PermissionDecision{}, err
		}
		if ok {
			return AllowDecision(), nil
		}
		return DenyDecision("permission denied for tool " + toolName), nil
	}
}

// ResolveAskWithDeps applies [ExecutionDeps.AskResolver] or headless default (deny).
func ResolveAskWithDeps(ctx context.Context, deps ExecutionDeps, toolName, toolUseID string, input json.RawMessage, prompt string) (PermissionDecision, error) {
	if deps.AskResolver != nil {
		return deps.AskResolver(ctx, toolName, toolUseID, input, prompt)
	}
	msg := prompt
	if msg == "" {
		msg = "permission ask required (no AskResolver; default deny)"
	}
	return DenyDecision(msg), nil
}
