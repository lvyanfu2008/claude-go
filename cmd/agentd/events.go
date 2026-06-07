package main

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/engine"
	"goc/gou/conversation"
	"goc/types"
)

// stdioEventHandler 将 engine.EventHandler 输出序列化为 NDJSON 写到 stdout。
type stdioEventHandler struct {
	enc *json.Encoder
}

func newStdioEventHandler(enc *json.Encoder) *stdioEventHandler {
	return &stdioEventHandler{enc: enc}
}

func (h *stdioEventHandler) sendEvent(eventType engine.AgentEventType, payload any) {
	var raw json.RawMessage
	if payload != nil {
		switch v := payload.(type) {
		case json.RawMessage:
			raw = v
		default:
			b, _ := json.Marshal(v)
			raw = json.RawMessage(b)
		}
	}
	_ = h.enc.Encode(engine.AgentEvent{Type: eventType, Payload: raw})
}

func (h *stdioEventHandler) OnStateSnapshot(messages []types.Message, metadata engine.StateMetadata) {
	h.sendEvent(engine.EventTypeStateSnapshot, map[string]any{
		"messages": messages,
		"metadata": metadata,
	})
}

func (h *stdioEventHandler) OnStreamDelta(delta string) {
	h.sendEvent(engine.EventTypeStreamDelta, map[string]string{"delta": delta})
}

func (h *stdioEventHandler) OnStreamThinkingDelta(delta string) {
	h.sendEvent(engine.EventTypeStreamThinkingDelta, map[string]string{"delta": delta})
}

func (h *stdioEventHandler) OnToolUseStart(tool string, toolUseID string, input json.RawMessage) {
	var inputAny any = string(input)
	var parsed any
	if json.Unmarshal(input, &parsed) == nil {
		inputAny = parsed
	}
	h.sendEvent(engine.EventTypeToolUseStart, map[string]any{
		"tool":        tool,
		"tool_use_id": toolUseID,
		"input":       inputAny,
	})
}

func (h *stdioEventHandler) OnToolUseEnd(toolUseID string, output string, isError bool) {
	h.sendEvent(engine.EventTypeToolUseEnd, map[string]any{
		"tool_use_id": toolUseID,
		"output":      output,
		"is_error":    isError,
	})
}

func (h *stdioEventHandler) OnToolResult(toolUseID string, content json.RawMessage, isError bool) {
	h.sendEvent(engine.EventTypeToolResult, map[string]any{
		"tool_use_id": toolUseID,
		"content":     content,
		"is_error":    isError,
	})
}

func (h *stdioEventHandler) OnTurnDone(stopReason string) {
	h.sendEvent(engine.EventTypeTurnDone, map[string]string{"stop_reason": stopReason})
}

func (h *stdioEventHandler) OnAssistantMessage(msg types.Message) {
	h.sendEvent(engine.EventTypeAssistantMessage, msg)
}

func (h *stdioEventHandler) OnAgentProgress(agentID string, status string, message string) {
	h.sendEvent(engine.EventTypeAgentProgress, map[string]string{
		"agent_id": agentID,
		"status":   status,
		"message":  message,
	})
}

func (h *stdioEventHandler) OnPermissionAsk(tool string, toolUseID string, input json.RawMessage) {
	var inputAny any = string(input)
	var parsed any
	if json.Unmarshal(input, &parsed) == nil {
		inputAny = parsed
	}
	h.sendEvent(engine.EventTypePermissionAsk, map[string]any{
		"tool":        tool,
		"tool_use_id": toolUseID,
		"input":       inputAny,
	})
}

func (h *stdioEventHandler) OnErrorMessage(errMsg string) {
	h.sendEvent(engine.EventTypeError, map[string]string{"message": errMsg})
}

// stdioPermissionBridge 通过 stdin reader goroutine 的 channel 实现非阻塞权限询问。
type stdioPermissionBridge struct {
	responseCh chan engine.PermissionDecision
}

func (b *stdioPermissionBridge) AskPermission(ctx context.Context, toolName string, input json.RawMessage) (engine.PermissionDecision, error) {
	select {
	case decision := <-b.responseCh:
		return decision, nil
	case <-ctx.Done():
		return engine.PermissionDecision{Allow: false, Reason: "cancelled"}, ctx.Err()
	}
}

// newAgentdSubmitFunc 创建 agentd 版本的 SubmitFunc。
func newAgentdSubmitFunc(cwd, sessionID string, events engine.EventHandler, perms engine.PermissionBridge) engine.SubmitFunc {
	return func(ctx context.Context, text string, store *conversation.Store, evt engine.EventHandler, prm engine.PermissionBridge) error {
		evt.OnStreamDelta(fmt.Sprintf("agentd received input (%d chars): %s\n", len(text), truncateText(text, 80)))

		// TODO: 接入完整的 ProcessUserInput + Query 流
		// 当前为简化版本，仅 echo 输入内容。
		// 完整实现需要：
		//   1. 构建 ProcessUserInputParams（注入 commands/tools/MCP 上下文）
		//   2. 调用 processuserinput.ProcessUserInput
		//   3. 构建 QueryParams（含 system prompt、user context 等）
		//   4. 调用 query.Query
		//   5. 将 yield 转为 EventHandler 调用
		//   6. 处理工具执行和权限

		store.AppendMessage(types.Message{
			Type:    types.MessageTypeUser,
			Content: mustMarshal([]map[string]string{{"type": "text", "text": text}}),
		})

		evt.OnTurnDone("completed")
		return nil
	}
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
