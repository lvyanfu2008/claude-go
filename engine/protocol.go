package engine

import "encoding/json"

// GatewayMessageType 是 Gateway → Agent 的消息类型。
type GatewayMessageType string

const (
	MsgTypeUserMessage        GatewayMessageType = "user_message"
	MsgTypePermissionResponse GatewayMessageType = "permission_response"
	MsgTypeResume             GatewayMessageType = "resume"
	MsgTypeAbort              GatewayMessageType = "abort"
)

// GatewayMessage 是 Gateway 发给 Agent 的请求。
type GatewayMessage struct {
	Type    GatewayMessageType `json:"type"`
	Payload json.RawMessage    `json:"payload,omitempty"`
}

// UserMessagePayload 用户提交的输入。
type UserMessagePayload struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Mode      string `json:"mode,omitempty"`
}

// PermissionResponsePayload 用户对权限询问的回复。
type PermissionResponsePayload struct {
	ToolUseID string `json:"tool_use_id"`
	Decision  string `json:"decision"` // "allow" | "deny"
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
