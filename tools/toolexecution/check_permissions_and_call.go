package toolexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"goc/internal/toolvalidator"
	toolregistry "goc/tools/tool"
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

	// Backfill observable input before hooks/observers see it (TS: query.ts L996, toolExecution.ts L827).
	if inputMap := make(map[string]any); json.Unmarshal(input, &inputMap) == nil {
		toolregistry.BackfillObservableInput(tool.Name(), inputMap)
		if backfilled, err := json.Marshal(inputMap); err == nil {
			input = backfilled
		}
	}

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
	effectiveInput := input
	switch dec.Behavior {
	case PermissionAllow:
		allowProceed = true
		if dec.UpdatedInput != nil {
			effectiveInput = dec.UpdatedInput
		}
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
		if final.UpdatedInput != nil {
			effectiveInput = final.UpdatedInput
		}
	default:
		return nil, fmt.Errorf("toolexecution: unknown permission behavior %q", dec.Behavior)
	}
	if !allowProceed {
		return nil, fmt.Errorf("toolexecution: internal permission state")
	}
	return finishCheckPermissionsWithToolCall(ctx, deps, tool, toolUseID, effectiveInput, tcxUse, canUseTool, assistant)
}

// finishCheckPermissionsWithToolCall runs [ExecutionDeps.InvokeTool] when set (same order as [RunToolUseChan]), else [Tool.Call], then one user row with tool_result.
// When [ExecutionDeps.PostToolUseHookRunner] is set, post-tool-use hooks are executed and
// their messages are appended to the result (TS parity: runPostToolUseHooks in toolHooks.ts).
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
	var toolResponse string
	var msgs []types.Message

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
		toolResponse = content
		um := syntheticToolMessageAfterInvoke(deps, tool.Name(), toolUseID, input, content, isErr, assistant.UUID)
		msgs = []types.Message{um}
	} else {
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
		toolResponse = body
		um := syntheticToolResult(deps, toolUseID, body, isErr, assistant.UUID)
		msgs = []types.Message{um}
	}

	// Run PostToolUse hooks if configured (TS parity: runPostToolUseHooks).
	if deps.PostToolUseHookRunner != nil {
		hookResults, err := deps.PostToolUseHookRunner(ctx, PostToolUseHookInput{
			ToolName:     tool.Name(),
			ToolUseID:    toolUseID,
			ToolInput:    input,
			ToolResponse: toolResponse,
		}, deps)
		if err == nil {
			for _, r := range hookResults {
				if len(r.Message) > 0 {
					var msg types.Message
					if json.Unmarshal(r.Message, &msg) == nil {
						msgs = append(msgs, msg)
					}
				}
			}
		}
	}

	return msgs, nil
}
