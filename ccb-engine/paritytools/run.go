package paritytools

import (
	"context"
	"errors"
)

// Run executes unconditional built-in tools for the Go parity runner.
// If the tool name is not handled here, it returns [ErrNotHandled].
func Run(ctx context.Context, name string, raw []byte, cfg Config) (string, bool, error) {
	switch name {
	case "NotebookEdit":
		return NotebookEditFromJSON(raw, cfg.Roots)
	case "TodoWrite":
		return TodoWriteFromJSON(raw, cfg)
	case "TaskOutput":
		return TaskOutputFromJSON(ctx, raw, cfg)
	case "TaskCreate":
		return TaskCreateFromJSON(ctx, raw, cfg)
	case "TaskGet":
		return TaskGetFromJSON(ctx, raw, cfg)
	case "TaskList":
		return TaskListFromJSON(ctx, raw, cfg)
	case "TaskUpdate":
		return TaskUpdateFromJSON(ctx, raw, cfg)
	case "TaskStop", "KillShell":
		return TaskStopFromJSON(raw, cfg)
	case "WebFetch":
		return WebFetchFromJSON(ctx, raw)
	case "WebSearch":
		return WebSearchFromJSON(ctx, raw)
	case "EnterPlanMode":
		return EnterPlanModeFromJSON(raw, cfg)
	case "ExitPlanMode":
		return ExitPlanModeFromJSON(raw, cfg)
	case "AskUserQuestion":
		return AskUserQuestionFromJSON(raw, cfg)
	case "CronCreate":
		return CronCreateFromJSON(raw, cfg)
	case "CronDelete":
		return CronDeleteFromJSON(raw, cfg)
	case "CronList":
		return CronListFromJSON(raw, cfg)
	case "Agent":
		return AgentStubFromJSON(raw)
	case "SendMessage":
		return SendMessageStubFromJSON(raw)
	case "SendUserMessage", "Brief":
		return BriefFromJSON(raw)
	case "ListMcpResourcesTool":
		return ListMcpResourcesStub(raw)
	case "ReadMcpResourceTool":
		return ReadMcpResourceStub(raw)
	default:
		return "", false, ErrNotHandled
	}
}

// ErrNotHandled means the tool should be executed by another runner layer.
var ErrNotHandled = errors.New("paritytools: tool not handled")

// IsNotHandled reports whether err is [ErrNotHandled].
func IsNotHandled(err error) bool {
	return errors.Is(err, ErrNotHandled)
}
