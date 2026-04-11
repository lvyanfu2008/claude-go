package toolexecution

import (
	"context"
	"encoding/json"
	"errors"
	"iter"

	"goc/types"
)

// StreamedCheckPermissionsAndCallTool mirrors streamedCheckPermissionsAndCallTool (toolExecution.ts L495–573).
// Today it drains [CheckPermissionsAndCallTool]; on [ErrPipelineNotImplemented] it yields one explicit skeleton tool_result.
func StreamedCheckPermissionsAndCallTool(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input json.RawMessage,
	tcx *ToolUseContext,
	canUseTool CanUseToolFn,
	assistant AssistantMeta,
) iter.Seq2[MessageUpdate, error] {
	return func(yield func(MessageUpdate, error) bool) {
		deps := DepsFromContext(ctx)
		msgs, err := CheckPermissionsAndCallTool(ctx, tool, toolUseID, input, tcx, canUseTool, assistant)
		if err != nil {
			if errors.Is(err, ErrPipelineNotImplemented) {
				m := syntheticPipelineTODO(deps, toolUseID, assistant.UUID)
				yield(MessageUpdate{Message: &m}, nil)
				return
			}
			yield(MessageUpdate{}, err)
			return
		}
		for i := range msgs {
			mm := msgs[i]
			if !yield(MessageUpdate{Message: &mm}, nil) {
				return
			}
		}
	}
}

// CheckPermissionsAndCallTool mirrors checkPermissionsAndCallTool (toolExecution.ts L602+).
//
// TODO(toolExecution.ts L617+): zod/inputSchema validation, full runPreToolUseHooks parity, tool.call,
// progress stream, MCP branches, post-tool hooks, telemetry spans, …
func CheckPermissionsAndCallTool(
	ctx context.Context,
	tool Tool,
	toolUseID string,
	input json.RawMessage,
	tcx *ToolUseContext,
	canUseTool CanUseToolFn,
	assistant AssistantMeta,
) ([]types.Message, error) {
	deps := DepsFromContext(ctx)
	if err := RunPreToolUseHooks(ctx, deps, tool.Name(), toolUseID, input); err != nil {
		um := syntheticPreToolHookDenied(deps, toolUseID, assistant.UUID, err.Error())
		return []types.Message{um}, nil
	}
	if st, ok := tool.(interface{ InputSchemaAny() any }); ok {
		if err := ValidateInputAgainstSchema(tool.Name(), st.InputSchemaAny(), input); err != nil {
			um := syntheticPreToolHookDenied(deps, toolUseID, assistant.UUID, err.Error())
			return []types.Message{um}, nil
		}
	}
	tcxUse := tcx
	if tcxUse == nil {
		tcxUse = &ToolUseContext{}
	}
	if deps.ToolPermission != nil {
		merged := *tcxUse
		if merged.ToolPermission == nil {
			merged.ToolPermission = deps.ToolPermission
		}
		tcxUse = &merged
	}
	hookPerm := deps.PreToolHookPermission
	dec, _, err := ResolveHookPermissionDecision(ctx, ResolveHookPermissionInput{
		HookPermission: hookPerm,
		Tool:           tool,
		Input:          input,
		TCX:            tcxUse,
		ToolUseID:      toolUseID,
		Assistant:      assistant,
		QueryGate:      deps.QueryCanUseTool,
		LegacyGate:     canUseTool,
	})
	if err != nil {
		return nil, err
	}
	switch dec.Behavior {
	case PermissionAllow:
		return nil, ErrPipelineNotImplemented
	case PermissionDeny:
		msg := dec.Message
		if msg == "" {
			msg = "permission denied"
		}
		um := syntheticPreToolHookDenied(deps, toolUseID, assistant.UUID, msg)
		return []types.Message{um}, nil
	case PermissionAsk:
		final, err := ResolveAskWithDeps(ctx, deps, tool.Name(), toolUseID, input, dec.Message)
		if err != nil {
			return nil, err
		}
		if final.Behavior != PermissionAllow {
			msg := final.Message
			if msg == "" {
				msg = "permission denied"
			}
			um := syntheticPreToolHookDenied(deps, toolUseID, assistant.UUID, msg)
			return []types.Message{um}, nil
		}
		return nil, ErrPipelineNotImplemented
	default:
		return nil, ErrPipelineNotImplemented
	}
}
