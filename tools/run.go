package tools

import (
	"context"
	"errors"
	"strings"

	"goc/tools/localtools"
)

func availableMCPServersFromEnv() []string {
	raw := strings.TrimSpace(getenv("CLAUDE_CODE_AVAILABLE_MCP_SERVERS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

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
	case "PowerShell":
		return localtools.PowerShellFromJSON(ctx, raw, cfg.WorkDir)
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
		return RunAgentTool(raw, AgentRuntimeConfig{
			WorkDir:             cfg.WorkDir,
			ProjectRoot:         cfg.ProjectRoot,
			SessionID:           cfg.SessionID,
			TasksDir:            cfg.TasksDir(),
			AvailableMCPServers: availableMCPServersFromEnv(),
			Messages:            cfg.Messages,
			SystemPrompt:        cfg.SystemPrompt,
			TeamName:            cfg.TeamName,
			AgentName:           cfg.AgentName,
			AgentID:             cfg.AgentID,
		ProgressCallback:    cfg.ProgressCallback,
	})
	case "SendMessage":
		return RunSendMessageTool(raw, AgentRuntimeConfig{
			WorkDir:             cfg.WorkDir,
			ProjectRoot:         cfg.ProjectRoot,
			SessionID:           cfg.SessionID,
			TasksDir:            cfg.TasksDir(),
			AvailableMCPServers: availableMCPServersFromEnv(),
			TeamName:            cfg.TeamName,
			AgentName:           cfg.AgentName,
			AgentID:             cfg.AgentID,
		})
	case "SendUserMessage", "Brief":
		return BriefFromJSON(raw)
	case "ListMcpResourcesTool":
		return ListMcpResourcesFromJSON(raw)
	case "ReadMcpResourceTool":
		return ReadMcpResourceFromJSON(raw)
	case "echo_stub":
		return EchoStubFromJSON(raw)
	case "TestingPermission":
		return TestingPermissionFromJSON(raw)
	case "Sleep":
		return SleepFromJSON(ctx, raw)
	case "ListPeers":
		return ListPeersFromJSON(raw, cfg)
	case "VerifyPlanExecution":
		return VerifyPlanExecutionFromJSON(raw)
	case "OverflowTest":
		return OverflowTestFromJSON(raw)
	case "CtxInspect":
		return CtxInspectFromJSON(raw)
	case "TerminalCapture":
		return TerminalCaptureFromJSON(raw)
	case "LSP":
		return LSPFromJSON(raw)
	case "EnterWorktree":
		return EnterWorktreeFromJSON(raw, cfg)
	case "ExitWorktree":
		return ExitWorktreeFromJSON(raw, cfg)
	case "TeamCreate":
		return TeamCreateFromJSON(raw, cfg)
	case "TeamDelete":
		return TeamDeleteFromJSON(raw, cfg)
	case "TeamAddMember":
		return TeamAddMemberFromJSON(raw, cfg)
	case "TeamRemoveMember":
		return TeamRemoveMemberFromJSON(raw, cfg)
	case "Config":
		return ConfigFromJSON(raw, cfg)
	case "Tungsten":
		return TungstenFromJSON(raw)
	case "SuggestBackgroundPR":
		return SuggestBackgroundPRFromJSON(raw)
	case "WebBrowser":
		return WebBrowserFromJSON(raw)
	case "RemoteTrigger":
		return RemoteTriggerFromJSON(raw)
	case "Monitor":
		return MonitorFromJSON(ctx, raw, cfg)
	case "workflow":
		return WorkflowFromJSON(raw)
	case "Snip":
		return SnipFromJSON(raw)
	case "SendUserFile":
		return SendUserFileFromJSON(raw)
	case "PushNotification":
		return PushNotificationFromJSON(raw)
	case "SubscribePR":
		return SubscribePRFromJSON(raw)
	case "ReviewArtifact":
		return ReviewArtifactFromJSON(raw)
	default:
		return "", false, ErrNotHandled
	}
}

// ErrNotHandled means the tool should be executed by another runner layer.
var ErrNotHandled = errors.New("tools: tool not handled")

// IsNotHandled reports whether err is [ErrNotHandled].
func IsNotHandled(err error) bool {
	return errors.Is(err, ErrNotHandled)
}
