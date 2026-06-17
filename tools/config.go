package tools

import (
	"goc/appstate"
	"goc/types"
)

// Config is passed from [skilltools.ParityToolRunner] into unconditional tool runners.
type Config struct {
	Roots        []string
	WorkDir      string
	ProjectRoot  string
	SessionID    string
	AskAutoFirst bool // when true, AskUserQuestion picks the first option per question (gou-demo default)
	// Messages carries the parent conversation messages (needed for fork subagent).
	Messages []types.Message
	// SystemPrompt carries the parent's rendered system prompt parts (needed for fork subagent).
	SystemPrompt []string
	// Team identity for multi-agent teams.
	AgentName string
	AgentID   string
	TeamName  string
	// MainLoopModel is the parent session's model, passed through to AgentRuntimeConfig
	// for subagent model resolution via GetAgentModel().
	MainLoopModel string
	// ProgressCallback forwards agent progress messages in real time to the UI.
	ProgressCallback func(*types.Message)
	// OnAgentProgress forwards workflow/agent node progress (agentID, status, message).
	OnAgentProgress func(agentID, status, message string)
	// NotificationCallback is called when a background agent completes.
	NotificationCallback func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)
	// ToolPermission is the parent's permission context (deny/allow/ask rules).
	// When non-nil, it is propagated to child agents for bubble-mode permission enforcement.
	ToolPermission *types.ToolPermissionContextData
	// ToolUseID carries the current tool_use block ID from RunToolUseChan through InvokeTool
	// so that child agent progress messages can be associated with the correct parent tool_use
	// via ParentToolUseID. Set by the InvokeTool closure in executeAgentWithOpts.
	ToolUseID string
	// AppStateStore is the session-scoped state store, used by EnterPlanMode/ExitPlanMode
	// to atomically update ToolPermissionContext and plan-mode flag fields.
	AppStateStore *appstate.Store
}
