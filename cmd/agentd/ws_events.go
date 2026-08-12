package main

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"

	"goc/engine"
	"goc/types"
)

// wsEventHandler 将 engine.EventHandler 输出序列化为 JSON 写到 WebSocket 帧。
type wsEventHandler struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func newWsEventHandler(conn *websocket.Conn) *wsEventHandler {
	return &wsEventHandler{conn: conn}
}

func (h *wsEventHandler) sendEvent(eventType engine.AgentEventType, payload any) {
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
	data, err := json.Marshal(engine.AgentEvent{Type: eventType, Payload: raw})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = h.conn.WriteMessage(websocket.TextMessage, data)
}

func (h *wsEventHandler) OnStateSnapshot(messages []types.Message, metadata engine.StateMetadata) {
	h.sendEvent(engine.EventTypeStateSnapshot, map[string]any{
		"messages": messages,
		"metadata": metadata,
	})
}

func (h *wsEventHandler) OnStreamDelta(delta string) {
	h.sendEvent(engine.EventTypeStreamDelta, map[string]string{"delta": delta})
}

func (h *wsEventHandler) OnStreamThinkingDelta(delta string) {
	h.sendEvent(engine.EventTypeStreamThinkingDelta, map[string]string{"delta": delta})
}

func (h *wsEventHandler) OnToolUseStart(tool string, toolUseID string, input json.RawMessage) {
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

func (h *wsEventHandler) OnToolUseEnd(toolUseID string, output string, isError bool) {
	h.sendEvent(engine.EventTypeToolUseEnd, map[string]any{
		"tool_use_id": toolUseID,
		"output":      output,
		"is_error":    isError,
	})
}

func (h *wsEventHandler) OnToolResult(toolUseID string, content json.RawMessage, isError bool) {
	h.sendEvent(engine.EventTypeToolResult, map[string]any{
		"tool_use_id": toolUseID,
		"content":     content,
		"is_error":    isError,
	})
}

func (h *wsEventHandler) OnTurnDone(stopReason string) {
	h.sendEvent(engine.EventTypeTurnDone, map[string]string{"stop_reason": stopReason})
}

func (h *wsEventHandler) OnAssistantMessage(msg types.Message) {
	h.sendEvent(engine.EventTypeAssistantMessage, msg)
}

func (h *wsEventHandler) OnAgentProgress(agentID string, status string, message string) {
	h.sendEvent(engine.EventTypeAgentProgress, map[string]string{
		"agent_id": agentID,
		"status":   status,
		"message":  message,
	})
}

func (h *wsEventHandler) OnPermissionAsk(tool string, toolUseID string, input json.RawMessage) {
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

func (h *wsEventHandler) OnErrorMessage(errMsg string) {
	h.sendEvent(engine.EventTypeError, map[string]string{"message": errMsg})
}

func (h *wsEventHandler) OnCommandsList(commands []types.Command) {
	h.sendEvent(engine.EventTypeCommandsList, map[string]any{
		"commands": commands,
	})
}

// wsPermissionBridge 通过 per-conn channel 实现权限询问(与 stdioPermissionBridge 同构)。
type wsPermissionBridge struct {
	responseCh chan engine.PermissionDecision
}

func (b *wsPermissionBridge) AskPermission(ctx context.Context, toolName string, input json.RawMessage) (engine.PermissionDecision, error) {
	select {
	case decision := <-b.responseCh:
		return decision, nil
	case <-ctx.Done():
		return engine.PermissionDecision{Allow: false, Reason: "cancelled"}, ctx.Err()
	}
}
