package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"goc/types"

	"goc/engine"
	"goc/gou/conversation"
	"goc/sessiontranscript"
	"goc/tools/toolexecution"
	"goc/types"
)

// bubbleTeaEventHandler 将 engine.EventHandler 调用转为 tea.Msg 发送到程序队列。
type bubbleTeaEventHandler struct {
	program *tea.Program
	model   *Model
}

func newBubbleTeaEventHandler(p *tea.Program, m *Model) *bubbleTeaEventHandler {
	return &bubbleTeaEventHandler{program: p, model: m}
}

func (h *bubbleTeaEventHandler) sendTyped(msg tea.Msg) {
	if h.program != nil {
		h.program.Send(msg)
	}
}

func (h *bubbleTeaEventHandler) OnStateSnapshot(messages []types.Message, metadata engine.StateMetadata) {
	h.model.store.Messages = messages
	if metadata.SessionID != "" {
		h.model.store.ConversationID = metadata.SessionID
	}
	if metadata.PermissionMode != "" {
		h.model.permissionMode = types.PermissionMode(metadata.PermissionMode)
	}
	if metadata.MainLoopModel != "" {
		h.model.lastMainLoopModel = metadata.MainLoopModel
	}
	h.sendTyped(tea.ClearScreen())
}

func (h *bubbleTeaEventHandler) OnStreamDelta(delta string) {
	h.model.store.AppendStreamingChunk(delta)
}

func (h *bubbleTeaEventHandler) OnStreamThinkingDelta(delta string) {
	h.model.store.AppendStreamingThinkingChunk(delta)
}

func (h *bubbleTeaEventHandler) OnToolUseStart(tool string, toolUseID string, input json.RawMessage) {
	// 在 Orchestrator 模式下，tool_use 由 submitFn 通过 query yield 驱动，
	// 直接更新 store 即可，不需要额外的 tea.Msg
}

func (h *bubbleTeaEventHandler) OnToolUseEnd(toolUseID string, output string, isError bool) {
	// 同上，由 query yield 驱动
}

func (h *bubbleTeaEventHandler) OnToolResult(toolUseID string, content json.RawMessage, isError bool) {
	// 同上
}

func (h *bubbleTeaEventHandler) OnTurnDone(stopReason string) {
	h.model.store.ClearStreaming()
	h.model.store.ClearStreamingToolUses()
	h.sendTyped(gouQueryDoneMsg{})
}

func (h *bubbleTeaEventHandler) OnAgentProgress(agentID string, status string, message string) {
	now := time.Now()
	h.sendTyped(AgentProgressMsg{
		AgentID: agentID,
		Progress: &AgentTaskProgress{
			LastActivity:     &now,
			LastActivityDesc: message,
			Summary:          status,
		},
	})
}

func (h *bubbleTeaEventHandler) OnAssistantMessage(msg types.Message) {
	h.model.store.AppendMessage(msg)
	h.sendTyped(gouQueryYieldMsg{Message: msg})
}

func (h *bubbleTeaEventHandler) OnErrorMessage(errMsg string) {
	h.sendTyped(gouQueryDoneMsg{Err: fmt.Errorf("%s", errMsg)})
}

func (h *bubbleTeaEventHandler) OnCommandsList(commands []types.Command) {
	// No-op: Bubble Tea TUI loads commands via its own slash picker.
}

// bubbleTeaPermissionBridge 将 engine.PermissionBridge 接入 Bubble Tea 的弹窗机制。
type bubbleTeaPermissionBridge struct {
	model *Model
}

func newBubbleTeaPermissionBridge(m *Model) *bubbleTeaPermissionBridge {
	return &bubbleTeaPermissionBridge{model: m}
}

func (b *bubbleTeaPermissionBridge) AskPermission(ctx context.Context, toolName string, input json.RawMessage) (engine.PermissionDecision, error) {
	ch := make(chan permissionAskReply, 1)

	inputStr := ""
	if len(input) > 0 {
		inputStr = string(input)
	}

	b.model.permModal.Activate(toolName, inputStr, ch)

	select {
	case reply := <-ch:
		allow := reply.dec.Behavior == toolexecution.PermissionAllow
		reason := reply.dec.Message
		return engine.PermissionDecision{Allow: allow, Reason: reason}, nil
	case <-ctx.Done():
		return engine.PermissionDecision{Allow: false, Reason: "cancelled"}, ctx.Err()
	}
}

// newBubbleTeaSubmitFn 创建 Bubble Tea 模式的 SubmitFunc。
// 当前为简化版本，用于验证 Orchestrator 集成。
// TODO: 覆盖完整的 gouSubmitFromPromptText 逻辑（ProcessUserInput → ApplyBaseResult → Query）。
func newBubbleTeaSubmitFn(m *Model) engine.SubmitFunc {
	return func(ctx context.Context, text string, store *conversation.Store, events engine.EventHandler, perms engine.PermissionBridge) error {
		msg := types.Message{
			Type:    types.MessageTypeUser,
			UUID:    sessiontranscript.NewUUID(),
		}
		content, _ := json.Marshal([]map[string]string{{"type": "text", "text": text}})
		msg.Content = content
		store.AppendMessage(msg)
		events.OnStateSnapshot(store.Messages, engine.StateMetadata{})

		assistantContent, _ := json.Marshal([]map[string]string{{"type": "text", "text": "Orchestrator received: " + text}})
		assistantMsg := types.Message{
			Type:    types.MessageTypeAssistant,
			Content: assistantContent,
		}
		events.OnAssistantMessage(assistantMsg)
		events.OnTurnDone("completed")
		return nil
	}
}

// mustMarshalJSON 是 JSON 序列化辅助函数，忽略错误。
func mustMarshalJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
