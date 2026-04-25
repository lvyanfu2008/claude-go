package toolexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"goc/internal/toolvalidator"
	"goc/types"
)

// StreamedCheckPermissionsAndCallTool mirrors streamedCheckPermissionsAndCallTool (toolExecution.ts L495–573).
// It drains [CheckPermissionsAndCallTool]; on [ErrPipelineNotImplemented] it yields one explicit skeleton tool_result (reserved for partial parity paths).
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

// CheckPermissionsAndCallTool mirrors checkPermissionsAndCallTool (toolExecution.ts L602+) for the headless subset:
// pre-tool hook, optional JSON schema validation, hook permission resolution, then [InvokeToolFunc] or [Tool.Call] and a synthetic user row with tool_result.
//
// TODO(toolExecution.ts): full hook parity, progress stream, MCP branches, post-tool hooks, telemetry spans, …
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
		if err := toolvalidator.ValidateInput(tool.Name(), st.InputSchemaAny(), input); err != nil {
			um := syntheticInputValidationError(deps, toolUseID, assistant.UUID, tool.Name(), err)
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
	if tcxUse.BashSandboxRule1b == nil {
		if b := bashSandboxRule1bFromExecutionDeps(deps); b != nil {
			merged := *tcxUse
			merged.BashSandboxRule1b = b
			tcxUse = &merged
		}
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
	allowProceed := false
	switch dec.Behavior {
	case PermissionAllow:
		allowProceed = true
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
		allowProceed = true
	default:
		return nil, fmt.Errorf("toolexecution: unknown permission behavior %q", dec.Behavior)
	}
	if !allowProceed {
		return nil, fmt.Errorf("toolexecution: internal permission state")
	}
	return finishCheckPermissionsWithToolCall(ctx, deps, tool, toolUseID, input, tcxUse, canUseTool, assistant)
}

// finishCheckPermissionsWithToolCall runs [ExecutionDeps.InvokeTool] when set (same order as [RunToolUseChan]), else [Tool.Call], then one user row with tool_result.
func finishCheckPermissionsWithToolCall(
	ctx context.Context,
	deps ExecutionDeps,
	tool Tool,
	toolUseID string,
	input json.RawMessage,
	tcxUse *ToolUseContext,
	canUseTool CanUseToolFn,
	assistant AssistantMeta,
) ([]types.Message, error) {
	if deps.InvokeTool != nil {
		content, isErr, ierr := deps.InvokeTool(ctx, tool.Name(), toolUseID, input)
		if ctx.Err() != nil {
			um := syntheticAborted(deps, toolUseID, assistant.UUID)
			return []types.Message{um}, nil
		}
		if ierr != nil {
			um := syntheticToolResult(deps, toolUseID, ierr.Error(), true, assistant.UUID)
			return []types.Message{um}, nil
		}
		um := syntheticToolMessageAfterInvoke(deps, tool.Name(), toolUseID, input, content, isErr, assistant.UUID)
		return []types.Message{um}, nil
	}
	res, err := tool.Call(ctx, toolUseID, input, tcxUse, canUseTool, assistant, nil)
	if ctx.Err() != nil {
		um := syntheticAborted(deps, toolUseID, assistant.UUID)
		return []types.Message{um}, nil
	}
	if err != nil {
		um := syntheticToolResult(deps, toolUseID, err.Error(), true, assistant.UUID)
		return []types.Message{um}, nil
	}
	body, isErr := toolRunResultString(res)
	um := syntheticToolResult(deps, toolUseID, body, isErr, assistant.UUID)
	return []types.Message{um}, nil
}
