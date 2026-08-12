package engine

import "encoding/json"

// GatewayMessageType 是 Gateway → Agent 的消息类型。
type GatewayMessageType string

const (
	MsgTypeUserMessage        GatewayMessageType = "user_message"
	MsgTypePermissionResponse GatewayMessageType = "permission_response"
	MsgTypeResume             GatewayMessageType = "resume"
	MsgTypeAbort              GatewayMessageType = "abort"
	// MsgTypeSetCwd 客户端上报当前工作目录(WS 模式下前端把项目目录发给 agentd,
	// agentd 用它作为会话工作目录,而不是全局 agentd 进程的启动目录)。
	MsgTypeSetCwd GatewayMessageType = "set_cwd"
)

// GatewayMessage 是 Gateway 发给 Agent 的请求。
type GatewayMessage struct {
	Type    GatewayMessageType `json:"type"`
	Payload json.RawMessage    `json:"payload,omitempty"`
}

// SetCwdPayload 客户端上报的工作目录。
type SetCwdPayload struct {
	Cwd string `json:"cwd"`
}

// UserMessagePayload 用户提交的输入。
type UserMessagePayload struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Mode      string `json:"mode,omitempty"`
}

// PermissionResponsePayload 用户对权限询问的回复。
type PermissionResponsePayload struct {
	ToolUseID    string          `json:"tool_use_id"`
	Decision     string          `json:"decision"` // "allow" | "deny"
	UpdatedInput json.RawMessage `json:"updated_input,omitempty"`
}

// ResumePayload 恢复已有会话。
type ResumePayload struct {
	SessionID string `json:"session_id"`
}

// AgentEventType 是 Agent → Gateway 的事件类型。
type AgentEventType string

const (
	EventTypeStateSnapshot       AgentEventType = "state_snapshot"
	EventTypeStreamDelta         AgentEventType = "stream_delta"
	EventTypeStreamThinkingDelta AgentEventType = "stream_thinking_delta"
	EventTypeToolUseStart        AgentEventType = "tool_use_start"
	EventTypeToolUseEnd          AgentEventType = "tool_use_end"
	EventTypeToolResult          AgentEventType = "tool_result"
	EventTypePermissionAsk       AgentEventType = "permission_ask"
	EventTypeTurnDone            AgentEventType = "turn_done"
	EventTypeAgentProgress       AgentEventType = "agent_progress"
	EventTypeAssistantMessage    AgentEventType = "assistant_message"
	EventTypeError               AgentEventType = "error"
	EventTypeCommandsList        AgentEventType = "commands_list"
)

// AgentEvent 是 Agent 发给 Gateway 的事件。
type AgentEvent struct {
	Type    AgentEventType  `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
