// RunToolUseChan is the Go channel analogue of async function* runToolUse (toolExecution.ts).
// Order: permission ([ExecutionDeps.QueryCanUseTool]) → [ExecutionDeps.InvokeTool] (if set) →
// [ExecutionDeps.Registry] ([Tool.Call] + JSON schema validation) → unknown tool.
package toolexecution

import (
	"context"
	"encoding/json"
	"strings"

	"goc/tools/localtools"
	"goc/conversation-runtime/streamingtool"
	"goc/types"
)

// RunToolUseChan streams [streamingtool.ToolRunUpdate] for one tool_use (mirrors runToolUse yields).
// It closes the returned channel when finished.
func RunToolUseChan(
	parent context.Context,
	block streamingtool.ToolUseBlock,
	assistant types.Message,
	deps ExecutionDeps,
	toolAbort *streamingtool.AbortController,
) <-chan streamingtool.ToolRunUpdate {
	ch := make(chan streamingtool.ToolRunUpdate, 8)
	ctx, cancel := context.WithCancel(parent)
	if toolAbort != nil {
		toolAbort.OnAbortOnce(func(any) { cancel() })
	}

	go func() {
		defer close(ch)
		defer cancel()

		if ctx.Err() != nil {
			m := syntheticAborted(deps, block.ID, assistant.UUID)
			ch <- streamingtool.ToolRunUpdate{Message: &m}
			return
		}

		if deps.QueryCanUseTool != nil {
			dec, err := deps.QueryCanUseTool(ctx, block.Name, block.ID, block.Input)
			if err != nil {
				m := syntheticToolResult(deps, block.ID, err.Error(), true, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			}
			switch dec.Behavior {
			case PermissionDeny:
				msg := dec.Message
				if msg == "" {
					msg = "permission denied for tool " + block.Name
				}
				m := syntheticToolResult(deps, block.ID, msg, true, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			case PermissionAsk:
				final, err := ResolveAskWithDeps(ctx, deps, block.Name, block.ID, block.Input, dec.Message)
				if err != nil {
					m := syntheticToolResult(deps, block.ID, err.Error(), true, assistant.UUID)
					ch <- streamingtool.ToolRunUpdate{Message: &m}
					return
				}
				if final.Behavior != PermissionAllow {
					msg := final.Message
					if msg == "" {
						msg = "permission denied"
					}
					m := syntheticToolResult(deps, block.ID, msg, true, assistant.UUID)
					ch <- streamingtool.ToolRunUpdate{Message: &m}
					return
				}
			case PermissionAllow:
				// continue to InvokeTool / Registry
			}
		}

		if applyRuleBasedDecisionInRun(ctx, deps, block.Name, block.ID, block.Input, assistant.UUID, ch) {
			return
		}

		if deps.InvokeTool != nil {
			content, isErr, err := deps.InvokeTool(ctx, block.Name, block.ID, block.Input)
			if ctx.Err() != nil {
				m := syntheticAborted(deps, block.ID, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			}
			if err != nil {
				content = err.Error()
				isErr = true
			}
			m := syntheticToolMessageAfterInvoke(deps, block.Name, block.ID, block.Input, content, isErr, assistant.UUID)
			ch <- streamingtool.ToolRunUpdate{Message: &m}
			return
		}

		if deps.Registry != nil {
			tool, ok := deps.Registry.FindToolByName(block.Name)
			if !ok {
				m := syntheticUnknownTool(deps, block.Name, block.ID, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			}
			tcx := &ToolUseContext{
				ToolPermission:    deps.ToolPermission,
				BashSandboxRule1b: bashSandboxRule1bFromExecutionDeps(deps),
			}
			res, err := tool.Call(ctx, block.ID, block.Input, tcx, nil, AssistantMeta{UUID: assistant.UUID}, nil)
			if ctx.Err() != nil {
				m := syntheticAborted(deps, block.ID, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			}
			if err != nil {
				m := syntheticToolResult(deps, block.ID, err.Error(), true, assistant.UUID)
				ch <- streamingtool.ToolRunUpdate{Message: &m}
				return
			}
			content, isErr := toolRunResultString(res)
			m := syntheticToolResult(deps, block.ID, content, isErr, assistant.UUID)
			ch <- streamingtool.ToolRunUpdate{Message: &m}
			return
		}

		m := syntheticUnknownTool(deps, block.Name, block.ID, assistant.UUID)
		ch <- streamingtool.ToolRunUpdate{Message: &m}
	}()

	return ch
}

// applyRuleBasedDecisionInRun enforces permissions.ts 1a–1b (incl. Bash 1b sandbox whole-tool ask bypass) after the query gate.
// Returns true when a terminal tool_result was sent on ch.
func applyRuleBasedDecisionInRun(
	ctx context.Context,
	deps ExecutionDeps,
	toolName, toolUseID string,
	input json.RawMessage,
	assistantUUID string,
	ch chan<- streamingtool.ToolRunUpdate,
) bool {
	tcx := &ToolUseContext{
		ToolPermission:    deps.ToolPermission,
		BashSandboxRule1b: bashSandboxRule1bFromExecutionDeps(deps),
	}
	var rd *PermissionDecision
	if deps.Registry != nil {
		if tool, ok := deps.Registry.FindToolByName(toolName); ok {
			rd = CheckRuleBasedPermissions(ctx, tool, input, tcx)
		}
	}
	if rd == nil {
		rd = wholeToolAlwaysDenyAsk(toolName, input, deps.ToolPermission, bashSandboxRule1bFromExecutionDeps(deps))
	}
	if rd == nil {
		return false
	}
	switch rd.Behavior {
	case PermissionDeny:
		msg := rd.Message
		if msg == "" {
			msg = "permission denied for tool " + toolName
		}
		m := syntheticToolResult(deps, toolUseID, msg, true, assistantUUID)
		ch <- streamingtool.ToolRunUpdate{Message: &m}
		return true
	case PermissionAsk:
		final, err := ResolveAskWithDeps(ctx, deps, toolName, toolUseID, input, rd.Message)
		if err != nil {
			m := syntheticToolResult(deps, toolUseID, err.Error(), true, assistantUUID)
			ch <- streamingtool.ToolRunUpdate{Message: &m}
			return true
		}
		if final.Behavior != PermissionAllow {
			msg := final.Message
			if msg == "" {
				msg = "permission denied"
			}
			m := syntheticToolResult(deps, toolUseID, msg, true, assistantUUID)
			ch <- streamingtool.ToolRunUpdate{Message: &m}
			return true
		}
		return false
	default:
		return false
	}
}

func syntheticToolResult(deps ExecutionDeps, toolUseID, content string, isErr bool, assistantUUID string) types.Message {
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     content,
		"is_error":    isErr,
		"tool_use_id": toolUseID,
	}}, content, assistantUUID)
}

func syntheticToolResultMapped(deps ExecutionDeps, toolUseID, toolResultBlockContent, toolUseResultRaw string, isErr bool, assistantUUID string) types.Message {
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     toolResultBlockContent,
		"is_error":    isErr,
		"tool_use_id": toolUseID,
	}}, toolUseResultRaw, assistantUUID)
}

// syntheticToolMessageAfterInvoke mirrors toolExecution.ts addToolResult: tool_result.content
// comes from mapToolResultToToolResultBlockParam while toolUseResult stays the tool's native Output object.
func syntheticToolMessageAfterInvoke(deps ExecutionDeps, toolName, toolUseID string, input json.RawMessage, body string, isErr bool, assistantUUID string) types.Message {
	body = strings.TrimSpace(body)
	if !isErr && toolName == "Read" && body != "" && json.Valid([]byte(body)) {
		var probe struct {
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(body), &probe) == nil {
			switch probe.Type {
			case "text", "file_unchanged":
				opts := localtools.ReadToolResultMapOptsForToolInput(input, deps.ReadToolRoots, deps.ReadToolMemCWD, deps.MainLoopModel)
				mapped, mErr := localtools.MapReadToolResultToAssistantText(body, opts)
				if mErr != nil {
					return syntheticToolResult(deps, toolUseID, mErr.Error(), true, assistantUUID)
				}
				return syntheticToolResultMapped(deps, toolUseID, mapped, body, false, assistantUUID)
			}
		}
	}
	if !isErr && toolName == "Grep" && body != "" {
		if block, err := localtools.MapGrepToolOutputToToolResultContent(body); err == nil {
			return syntheticToolResultMapped(deps, toolUseID, block, body, false, assistantUUID)
		}
	}
	return syntheticToolResult(deps, toolUseID, body, isErr, assistantUUID)
}

func toolRunResultString(res *types.ToolRunResult) (content string, isErr bool) {
	if res == nil {
		return "{}", false
	}
	raw := res.Data
	if len(raw) == 0 {
		return "{}", false
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return "{}", false
	}
	// Pretty JSON for object/array; raw string otherwise
	var v any
	if json.Unmarshal(raw, &v) == nil {
		if b, err := json.Marshal(v); err == nil {
			return string(b), false
		}
	}
	return s, false
}
