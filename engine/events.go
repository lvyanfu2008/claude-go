package engine

import (
	"encoding/json"

	"goc/types"
)

// StateMetadata 是 state_snapshot 携带的会话元数据。
type StateMetadata struct {
	SessionID      string `json:"session_id"`
	PermissionMode string `json:"permission_mode,omitempty"`
	MainLoopModel  string `json:"main_loop_model,omitempty"`
}

// EventHandler 是引擎输出事件的接收者。
// 引擎产生事件，实现者决定如何消费。
// Bubble Tea 实现者将事件转为 tea.Msg 发到程序队列。
// Agentd 实现者将事件序列化为 NDJSON 写到 stdout。
type EventHandler interface {
	// OnStateSnapshot 全量状态快照（resume 或启动时下发）。
	OnStateSnapshot(messages []types.Message, metadata StateMetadata)

	// OnStreamDelta 逐字推送 LLM 文本输出（打字机效果）。
	OnStreamDelta(delta string)

	// OnStreamThinkingDelta 逐字推送思考过程。
	OnStreamThinkingDelta(delta string)

	// OnToolUseStart 工具调用开始。
	OnToolUseStart(tool string, toolUseID string, input json.RawMessage)

	// OnToolUseEnd 工具调用完成。
	OnToolUseEnd(toolUseID string, output string, isError bool)

	// OnToolResult 工具结果返回。
	OnToolResult(toolUseID string, content json.RawMessage, isError bool)

	// OnTurnDone 本轮查询结束。
	OnTurnDone(stopReason string)

	// OnAgentProgress 后台 agent 进度更新。
	OnAgentProgress(agentID string, status string, message string)

	// OnAssistantMessage 完整 assistant 消息就绪（text + tool_uses 已组装好）。
	OnAssistantMessage(msg types.Message)

	// OnErrorMessage 错误信息输出。
	OnErrorMessage(errMsg string)
}
