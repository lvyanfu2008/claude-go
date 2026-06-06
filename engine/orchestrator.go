package engine

import (
	"context"
	"encoding/json"
	"sync"

	"goc/gou/conversation"
	"goc/sessiontranscript"
	"goc/types"
)

// pendingInput 是排队等待处理的用户输入。
type pendingInput struct {
	text string
	ctx  context.Context
}

// Orchestrator 管理对话核心循环。
//
// 它不关心 UI —— 输出通过 EventHandler，输入通过 SubmitInput。
// 它不关心 ProcessUserInput/Query 的具体实现 —— 调用方通过 SubmitFunc 注入。
//
// Orchestrator 负责：
//   - pending queue（FIFO，前一个 turn 完成后自动处理下一个）
//   - busy 状态管理
//   - context 管理（Abort 取消当前 query）
//   - 消息持久化（RecordTranscript）
type Orchestrator struct {
	store      *conversation.Store
	transcript *sessiontranscript.Store
	events     EventHandler

	// submitFn 是调用方提供的提交函数。
	// 调用方负责创建 ProcessUserInputParams、调用 ProcessUserInput、
	// 构建 QueryParams、调用 Query，以及将 yield 转发给 EventHandler。
	// Orchestrator 只负责执行 submitFn 时的排队和并发控制。
	submitFn SubmitFunc

	// permissionBridge 是可选的依赖注入，供 submitFn 使用。
	permissionBridge PermissionBridge

	mu            sync.Mutex
	pendingInputs []pendingInput
	busy          bool

	// cancelFn 取消当前正在运行的 query（调用 submitFn 的 context）。
	cancelFn context.CancelFunc
}

// SubmitFunc 是调用方注入的实际提交逻辑。
// ctx 会被 Orchestrator 管理（Abort 时会 cancel 这个 ctx）。
// store, events, permissionBridge 可以自由使用。
// 函数返回后，Orchestrator 会检查 pending queue。
type SubmitFunc func(ctx context.Context, text string, store *conversation.Store, events EventHandler, perms PermissionBridge) error

// NewOrchestrator 创建 Orchestrator。
func NewOrchestrator(
	store *conversation.Store,
	transcript *sessiontranscript.Store,
	events EventHandler,
	submitFn SubmitFunc,
	permissionBridge PermissionBridge,
) *Orchestrator {
	return &Orchestrator{
		store:            store,
		transcript:       transcript,
		events:           events,
		submitFn:         submitFn,
		permissionBridge: permissionBridge,
	}
}

// SubmitInput 提交用户输入。如果当前正在处理 query，输入会被排队。
// 排队是 FIFO 的——前一个 turn 完成后自动处理下一个。
func (o *Orchestrator) SubmitInput(ctx context.Context, text string) {
	o.mu.Lock()
	if o.busy {
		o.pendingInputs = append(o.pendingInputs, pendingInput{text: text, ctx: ctx})
		o.mu.Unlock()
		return
	}
	o.mu.Unlock()
	o.executeInput(ctx, text)
}

// executeInput 执行一次完整的对话 turn。
// 调用 submitFn，完成后检查 pending queue。
func (o *Orchestrator) executeInput(ctx context.Context, text string) {
	o.mu.Lock()
	o.busy = true
	turnCtx, cancel := context.WithCancel(ctx)
	o.cancelFn = cancel
	o.mu.Unlock()

	var err error
	func() {
		defer cancel()
		err = o.submitFn(turnCtx, text, o.store, o.events, o.permissionBridge)
	}()

	if err != nil {
		o.events.OnErrorMessage(err.Error())
	}

	// 持久化
	o.maybeRecordTranscript()

	o.mu.Lock()
	o.busy = false
	o.cancelFn = nil
	o.mu.Unlock()

	o.drainQueue()
}

// drainQueue 检查队列，处理下一条排队消息。
func (o *Orchestrator) drainQueue() {
	o.mu.Lock()
	if len(o.pendingInputs) == 0 {
		o.mu.Unlock()
		return
	}
	next := o.pendingInputs[0]
	o.pendingInputs = o.pendingInputs[1:]
	o.mu.Unlock()

	o.executeInput(next.ctx, next.text)
}

// Abort 取消当前正在运行的 query 并清空排队。
func (o *Orchestrator) Abort(_ context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.cancelFn != nil {
		o.cancelFn()
		o.cancelFn = nil
	}
	o.pendingInputs = nil
	o.busy = false
}

// IsBusy 返回当前是否正在处理 query。
func (o *Orchestrator) IsBusy() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.busy
}

// GetMessages 返回当前会话的消息列表副本。
func (o *Orchestrator) GetMessages() []types.Message {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]types.Message, len(o.store.Messages))
	copy(out, o.store.Messages)
	return out
}

// maybeRecordTranscript 持久化当前消息列表。
func (o *Orchestrator) maybeRecordTranscript() {
	if o.transcript == nil {
		return
	}
	msgs := make([]types.Message, len(o.store.Messages))
	copy(msgs, o.store.Messages)
	_, _ = o.transcript.RecordTranscript(context.Background(), msgs, sessiontranscript.RecordOpts{AllMessages: msgs})
}

// OnMessage 将完整消息追加到 store 并触发事件。
// submitFn 在处理 query yield 时可调用此方法。
func OnMessage(store *conversation.Store, events EventHandler, msg types.Message) {
	store.Messages = append(store.Messages, msg)

	switch msg.Type {
	case types.MessageTypeAssistant:
		// 解析 content blocks 并逐个通知
		var blocks []json.RawMessage
		if err := json.Unmarshal([]byte(msg.Content), &blocks); err == nil {
			for _, raw := range blocks {
				var b struct {
					Type    string          `json:"type"`
					Text    string          `json:"text,omitempty"`
					Name    string          `json:"name,omitempty"`
					ID      string          `json:"id,omitempty"`
					Input   json.RawMessage `json:"input,omitempty"`
					Content json.RawMessage `json:"content,omitempty"`
				}
				if json.Unmarshal(raw, &b) != nil {
					continue
				}
				switch b.Type {
				case "text":
					events.OnStreamDelta(b.Text)
				case "tool_use":
					events.OnToolUseStart(b.Name, b.ID, b.Input)
				case "tool_result":
					events.OnToolResult(b.ID, b.Content, false)
				}
			}
		}
		events.OnAssistantMessage(msg)

	case types.MessageTypeUser:
		events.OnStateSnapshot(store.Messages, StateMetadata{})
	}
}
