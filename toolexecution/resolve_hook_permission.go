package toolexecution

// TS: src/services/tools/toolHooks.ts resolveHookPermissionDecision (L332–433).

import (
	"context"
	"encoding/json"
)

// toolRequiresUserInteraction is true when the tool opts into TS requiresUserInteraction.
type toolInteraction interface {
	RequiresUserInteraction() bool
}

func toolRequiresInteraction(tool Tool) bool {
	if t, ok := tool.(toolInteraction); ok {
		return t.RequiresUserInteraction()
	}
	return false
}

// ResolveHookPermissionInput carries the reduced state needed to mirror resolveHookPermissionDecision.
type ResolveHookPermissionInput struct {
	HookPermission *PermissionDecision
	Tool           Tool
	Input          json.RawMessage
	TCX            *ToolUseContext
	ToolUseID      string
	Assistant      AssistantMeta
	QueryGate      QueryCanUseToolFn
	LegacyGate     CanUseToolFn
}

func effectiveQueryGate(in ResolveHookPermissionInput) QueryCanUseToolFn {
	if in.QueryGate != nil {
		return in.QueryGate
	}
	if in.LegacyGate == nil || in.TCX == nil {
		return nil
	}
	tcx := in.TCX
	lg := in.LegacyGate
	return func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (PermissionDecision, error) {
		if err := lg(toolName, input, tcx); err != nil {
			return DenyDecision(err.Error()), nil
		}
		return AllowDecision(), nil
	}
}

// ResolveHookPermissionDecision mirrors resolveHookPermissionDecision (toolHooks.ts L332–433) for the headless subset.
func ResolveHookPermissionDecision(ctx context.Context, in ResolveHookPermissionInput) (dec PermissionDecision, outInput json.RawMessage, err error) {
	outInput = in.Input
	gate := effectiveQueryGate(in)
	tcx := in.TCX
	if tcx == nil {
		tcx = &ToolUseContext{}
	}

	hook := in.HookPermission
	tool := in.Tool

	if hook != nil && hook.Behavior == PermissionDeny {
		return *hook, outInput, nil
	}

	if hook != nil && hook.Behavior == PermissionAllow {
		requires := toolRequiresInteraction(tool)
		requireGate := tcx.RequireCanUseTool
		// TS: updatedInput on hook — Go hook path does not carry updatedInput yet; use in.Input.
		hookInput := outInput
		// Without hook updatedInput on the Go side yet, interactive tools still require the query gate.
		interactionSatisfied := false

		if (requires && !interactionSatisfied) || requireGate {
			if gate == nil {
				return DenyDecision("canUseTool required but no QueryCanUseTool/LegacyGate"), outInput, nil
			}
			d, e := gate(ctx, tool.Name(), in.ToolUseID, hookInput)
			return d, hookInput, e
		}

		rule := CheckRuleBasedPermissions(ctx, tool, hookInput, tcx)
		if rule == nil {
			return *hook, hookInput, nil
		}
		if rule.Behavior == PermissionDeny {
			return *rule, hookInput, nil
		}
		if rule.Behavior == PermissionAsk {
			if gate == nil {
				return DenyDecision("ask rule requires prompt but no gate"), hookInput, nil
			}
			d, e := gate(ctx, tool.Name(), in.ToolUseID, hookInput)
			return d, hookInput, e
		}
		return *hook, hookInput, nil
	}

	if hook != nil && hook.Behavior == PermissionAsk {
		if gate == nil {
			return DenyDecision(hook.Message), outInput, nil
		}
		d, e := gate(ctx, tool.Name(), in.ToolUseID, outInput)
		return d, outInput, e
	}

	if gate == nil {
		return AllowDecision(), outInput, nil
	}
	d, e := gate(ctx, tool.Name(), in.ToolUseID, outInput)
	return d, outInput, e
}
