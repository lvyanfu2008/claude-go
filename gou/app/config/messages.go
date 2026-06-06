package config

import (
	"encoding/json"

	"goc/conversation-runtime/query"
	"goc/types"
)

type QueryYieldMsg struct {
	Message types.Message
}

type StreamEventMsg struct {
	Raw json.RawMessage
}

type StreamingToolUsesMsg struct {
	Uses []query.StreamingToolUseLive
}

type QueryDoneMsg struct {
	Err error
}

type MemoryAppendMsg struct {
	Msg types.Message
}

type AgentRegisteredMsg struct {
	Task AgentTaskInfo
}

type AgentTaskInfo struct {
	ID     string
	Label  string
	Status string
}

type AgentProgressMsg struct {
	AgentID  string
	Progress AgentTaskProgress
}

type AgentTaskProgress struct {
	TokenCount int
}

type AgentCompletedMsg struct {
	AgentID string
	Status  string
	Result  string
}

type AgentTaskTickMsg struct{}

type CommandQueueNotifyMsg struct{}

type CompactPhaseMsg struct {
	Phase string
}

type SpinnerTickMsg struct{}

type ToolSummaryDelayTickMsg struct{}
