package streamingtool

import (
	"context"
	"encoding/json"
	"iter"
	"sync"

	"goc/types"
)

// StreamingToolExecutor mirrors the class StreamingToolExecutor in src/services/tools/StreamingToolExecutor.ts.
type StreamingToolExecutor struct {
	mu sync.Mutex

	findTool   func(name string) (ToolBehavior, bool)
	canUseTool any
	toolCtx    ToolUseContextPort
	runner     ToolRunner

	tools []*trackedTool

	hasErrored            bool
	erroredToolDescription string
	siblingAbort          *AbortController
	discarded             bool

	progressSig chan struct{}
}

type trackedTool struct {
	id                string
	block             ToolUseBlock
	assistantMessage  types.Message
	status            ToolStatus
	isConcurrencySafe bool
	done              chan struct{} // closed when collectResults finishes; nil if never started

	results           []types.Message
	pendingProgress   []types.Message
	contextModifiers  []func(ToolUseContextPort) ToolUseContextPort
}

// NewStreamingToolExecutor mirrors the TS constructor:
// new StreamingToolExecutor(toolDefinitions, canUseTool, toolUseContext).
func NewStreamingToolExecutor(
	findTool func(name string) (ToolBehavior, bool),
	canUseTool any,
	toolCtx ToolUseContextPort,
	runner ToolRunner,
) *StreamingToolExecutor {
	sibling := CreateChildAbortController(toolCtx.QueryAbort())
	return &StreamingToolExecutor{
		findTool:    findTool,
		canUseTool:  canUseTool,
		toolCtx:     toolCtx,
		runner:      runner,
		siblingAbort: sibling,
		progressSig: make(chan struct{}, 1),
	}
}

// Discard mirrors discard() in StreamingToolExecutor.ts.
func (e *StreamingToolExecutor) Discard() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.discarded = true
}

// AddTool mirrors addTool(block, assistantMessage).
func (e *StreamingToolExecutor) AddTool(block ToolUseBlock, assistantMessage types.Message) {
	e.mu.Lock()
	if e.findTool == nil {
		e.mu.Unlock()
		go e.processQueue()
		return
	}
	def, ok := e.findTool(block.Name)
	if !ok {
		msg := createUserMessage([]map[string]any{{
			"type":        "tool_result",
			"content":     `<tool_use_error>Error: No such tool available: ` + block.Name + `</tool_use_error>`,
			"is_error":    true,
			"tool_use_id": block.ID,
		}}, "Error: No such tool available: "+block.Name, assistantMessage.UUID)
		e.tools = append(e.tools, &trackedTool{
			id:                block.ID,
			block:             block,
			assistantMessage:  assistantMessage,
			status:            ToolCompleted,
			isConcurrencySafe: true,
			results:           []types.Message{msg},
			pendingProgress:   nil,
		})
		e.mu.Unlock()
		go e.processQueue()
		return
	}

	parsed, inputOK := def.InputOK(block.Input)
	isConc := false
	if inputOK {
		func() {
			defer func() { recover() }()
			isConc = def.IsConcurrencySafe(parsed)
		}()
	}
	e.tools = append(e.tools, &trackedTool{
		id:                block.ID,
		block:             block,
		assistantMessage:  assistantMessage,
		status:            ToolQueued,
		isConcurrencySafe: isConc,
		pendingProgress:   nil,
	})
	e.mu.Unlock()
	go e.processQueue()
}

func (e *StreamingToolExecutor) canExecuteTool(isConcurrencySafe bool) bool {
	executing := 0
	var allConc = true
	for _, t := range e.tools {
		if t.status == ToolExecuting {
			executing++
			if !t.isConcurrencySafe {
				allConc = false
			}
		}
	}
	return executing == 0 || (isConcurrencySafe && allConc)
}

func (e *StreamingToolExecutor) processQueue() {
	e.mu.Lock()
	for _, tool := range e.tools {
		if tool.status != ToolQueued {
			continue
		}
		if !e.canExecuteTool(tool.isConcurrencySafe) {
			if !tool.isConcurrencySafe {
				break
			}
			continue
		}
		tool.status = ToolExecuting
		tool.done = make(chan struct{})
		e.toolCtx.SetInProgressToolUseIDs(func(prev map[string]struct{}) map[string]struct{} {
			next := cloneSet(prev)
			next[tool.id] = struct{}{}
			return next
		})
		e.updateInterruptibleState()
		td := tool
		e.mu.Unlock()
		go func(t *trackedTool) {
			defer close(t.done)
			defer func() { go e.processQueue() }()
			e.collectResults(t)
		}(td)
		e.mu.Lock()
	}
	e.mu.Unlock()
}

func (e *StreamingToolExecutor) createSyntheticErrorMessage(
	toolUseID string,
	reason string,
	assistantMessage types.Message,
	siblingErroredToolDescription string,
) types.Message {
	if reason == "user_interrupted" {
		return createUserMessage([]map[string]any{{
			"type":        "tool_result",
			"content":     withMemoryCorrectionHint(rejectMessage),
			"is_error":    true,
			"tool_use_id": toolUseID,
		}}, "User rejected tool use", assistantMessage.UUID)
	}
	if reason == "streaming_fallback" {
		return createUserMessage([]map[string]any{{
			"type":        "tool_result",
			"content":     "<tool_use_error>Error: Streaming fallback - tool execution discarded</tool_use_error>",
			"is_error":    true,
			"tool_use_id": toolUseID,
		}}, "Streaming fallback - tool execution discarded", assistantMessage.UUID)
	}
	msg := "Cancelled: parallel tool call errored"
	if siblingErroredToolDescription != "" {
		msg = "Cancelled: parallel tool call " + siblingErroredToolDescription + " errored"
	}
	return createUserMessage([]map[string]any{{
		"type":        "tool_result",
		"content":     "<tool_use_error>" + msg + "</tool_use_error>",
		"is_error":    true,
		"tool_use_id": toolUseID,
	}}, msg, assistantMessage.UUID)
}

// getAbortReason mirrors getAbortReason (TS private). Caller must NOT hold e.mu.
func (e *StreamingToolExecutor) getAbortReason(tool *trackedTool) (reason string, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.getAbortReasonLocked(tool)
}

// getAbortReasonLocked is the TS body; caller must hold e.mu.
func (e *StreamingToolExecutor) getAbortReasonLocked(tool *trackedTool) (reason string, ok bool) {
	_ = tool
	if e.discarded {
		return "streaming_fallback", true
	}
	if e.hasErrored {
		return "sibling_error", true
	}
	sig := e.toolCtx.QueryAbort().Signal()
	if sig.Aborted() {
		if sig.Reason() == AbortReasonInterrupt {
			if e.getToolInterruptBehavior(tool) == "cancel" {
				return "user_interrupted", true
			}
			return "", false
		}
		return "user_interrupted", true
	}
	return "", false
}

func (e *StreamingToolExecutor) getToolInterruptBehavior(tool *trackedTool) string {
	if e.findTool == nil {
		return "block"
	}
	def, ok := e.findTool(tool.block.Name)
	if !ok || def == nil {
		return "block"
	}
	var b string
	func() {
		defer func() { recover() }()
		b = def.InterruptBehavior()
	}()
	if b == "cancel" {
		return "cancel"
	}
	return "block"
}

func (e *StreamingToolExecutor) getToolDescription(tool *trackedTool) string {
	var in map[string]any
	_ = json.Unmarshal(tool.block.Input, &in)
	var summary string
	if v, ok := in["command"].(string); ok {
		summary = v
	} else if v, ok := in["file_path"].(string); ok {
		summary = v
	} else if v, ok := in["pattern"].(string); ok {
		summary = v
	}
	if summary != "" {
		if len(summary) > 40 {
			summary = summary[:40] + "\u2026"
		}
		return tool.block.Name + "(" + summary + ")"
	}
	return tool.block.Name
}

// updateInterruptibleState mirrors updateInterruptibleState in StreamingToolExecutor.ts.
// Caller must hold e.mu.
func (e *StreamingToolExecutor) updateInterruptibleState() {
	executing := e.toolsFiltered(func(t *trackedTool) bool { return t.status == ToolExecuting })
	if len(executing) == 0 {
		e.toolCtx.SetHasInterruptibleToolInProgress(false)
		return
	}
	allCancel := true
	for _, t := range executing {
		if e.getToolInterruptBehavior(t) != "cancel" {
			allCancel = false
			break
		}
	}
	e.toolCtx.SetHasInterruptibleToolInProgress(len(executing) > 0 && allCancel)
}

func (e *StreamingToolExecutor) toolsFiltered(pred func(*trackedTool) bool) []*trackedTool {
	var out []*trackedTool
	for _, t := range e.tools {
		if pred(t) {
			out = append(out, t)
		}
	}
	return out
}

func (e *StreamingToolExecutor) collectResults(tool *trackedTool) {
	e.mu.Lock()
	initialReason, initialHit := e.getAbortReasonLocked(tool)
	if initialHit {
		sd := ""
		if initialReason == "sibling_error" {
			sd = e.erroredToolDescription
		}
		m := e.createSyntheticErrorMessage(tool.id, initialReason, tool.assistantMessage, sd)
		tool.results = []types.Message{m}
		tool.status = ToolCompleted
		e.updateInterruptibleState()
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	toolAbort := CreateChildAbortController(e.siblingAbort)
	toolAbort.OnAbortOnce(func(reason any) {
		if reason == AbortReasonSiblingError {
			return
		}
		e.mu.Lock()
		d := e.discarded
		e.mu.Unlock()
		if d {
			return
		}
		if !e.toolCtx.QueryAbort().Signal().Aborted() {
			e.toolCtx.QueryAbort().Abort(reason)
		}
	})

	var messages []types.Message
	var contextModifiers []func(ToolUseContextPort) ToolUseContextPort
	thisToolErrored := false

	ch := e.runner.RunToolUpdates(tool.block, tool.assistantMessage, e.canUseTool, e.toolCtx, toolAbort)
	for upd := range ch {
		abortReason, abortHit := e.getAbortReason(tool)
		if abortHit && !thisToolErrored {
			sd := ""
			if abortReason == "sibling_error" {
				e.mu.Lock()
				sd = e.erroredToolDescription
				e.mu.Unlock()
			}
			m := e.createSyntheticErrorMessage(tool.id, abortReason, tool.assistantMessage, sd)
			messages = append(messages, m)
			break
		}

		if upd.Message != nil && isErrorToolResult(upd.Message) {
			thisToolErrored = true
			if tool.block.Name == BashToolName || tool.block.Name == BashZogToolName {
				e.mu.Lock()
				e.hasErrored = true
				e.erroredToolDescription = e.getToolDescription(tool)
				e.siblingAbort.Abort(AbortReasonSiblingError)
				e.mu.Unlock()
			}
		}

		if upd.Message != nil {
			if upd.Message.Type == types.MessageTypeProgress {
				e.mu.Lock()
				tool.pendingProgress = append(tool.pendingProgress, *upd.Message)
				e.mu.Unlock()
				select {
				case e.progressSig <- struct{}{}:
				default:
				}
			} else {
				m := *upd.Message
				messages = append(messages, m)
			}
		}
		if upd.ContextModifier != nil {
			contextModifiers = append(contextModifiers, upd.ContextModifier)
		}
	}

	e.mu.Lock()
	tool.results = messages
	tool.contextModifiers = contextModifiers
	tool.status = ToolCompleted
	e.updateInterruptibleState()
	if !tool.isConcurrencySafe && len(contextModifiers) > 0 {
		for _, mod := range contextModifiers {
			e.toolCtx = mod(e.toolCtx)
		}
	}
	e.mu.Unlock()
}

func isErrorToolResult(msg *types.Message) bool {
	if msg == nil || msg.Type != types.MessageTypeUser {
		return false
	}
	var env struct {
		Content []struct {
			Type    string `json:"type"`
			IsError bool   `json:"is_error"`
		} `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &env); err != nil {
		return false
	}
	for _, c := range env.Content {
		if c.Type == "tool_result" && c.IsError {
			return true
		}
	}
	return false
}

// GetCompletedResults mirrors *getCompletedResults() generator (non-blocking, one TS generator pass).
func (e *StreamingToolExecutor) GetCompletedResults() iter.Seq[MessageUpdate] {
	return func(yield func(MessageUpdate) bool) {
		batch := e.snapshotCompletedResults()
		for _, u := range batch {
			if !yield(u) {
				return
			}
		}
	}
}

func (e *StreamingToolExecutor) snapshotCompletedResults() []MessageUpdate {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.discarded {
		return nil
	}
	var out []MessageUpdate
	for _, tool := range e.tools {
		for len(tool.pendingProgress) > 0 {
			pm := tool.pendingProgress[0]
			tool.pendingProgress = tool.pendingProgress[1:]
			pmc := pm
			out = append(out, MessageUpdate{Message: &pmc, NewContext: e.toolCtx})
		}
		if tool.status == ToolYielded {
			continue
		}
		if tool.status == ToolCompleted && len(tool.results) > 0 {
			tool.status = ToolYielded
			for i := range tool.results {
				m := tool.results[i]
				mm := m
				out = append(out, MessageUpdate{Message: &mm, NewContext: e.toolCtx})
			}
			markToolUseAsComplete(e.toolCtx, tool.id)
		} else if tool.status == ToolExecuting && !tool.isConcurrencySafe {
			break
		}
	}
	return out
}

func (e *StreamingToolExecutor) hasPendingProgress() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tools {
		if len(t.pendingProgress) > 0 {
			return true
		}
	}
	return false
}

// RemainingResults mirrors async *getRemainingResults() (drains until all yielded).
func (e *StreamingToolExecutor) RemainingResults(ctx context.Context) iter.Seq2[MessageUpdate, error] {
	return func(yield func(MessageUpdate, error) bool) {
		for {
			select {
			case <-ctx.Done():
				yield(MessageUpdate{}, ctx.Err())
				return
			default:
			}
			if !e.hasUnfinishedTools() {
				break
			}
			e.processQueue()
			for u := range e.GetCompletedResults() {
				if !yield(u, nil) {
					return
				}
			}
			if e.hasExecutingTools() && !e.hasCompletedResults() && !e.hasPendingProgress() {
				e.waitAnyExecutingOrProgress(ctx)
			}
		}
		for u := range e.GetCompletedResults() {
			if !yield(u, nil) {
				return
			}
		}
	}
}

func (e *StreamingToolExecutor) hasCompletedResults() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tools {
		if t.status == ToolCompleted {
			return true
		}
	}
	return false
}

func (e *StreamingToolExecutor) hasExecutingTools() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tools {
		if t.status == ToolExecuting {
			return true
		}
	}
	return false
}

func (e *StreamingToolExecutor) hasUnfinishedTools() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tools {
		if t.status != ToolYielded {
			return true
		}
	}
	return false
}

func (e *StreamingToolExecutor) waitAnyExecutingOrProgress(ctx context.Context) {
	e.mu.Lock()
	var chans []<-chan struct{}
	for _, t := range e.tools {
		if t.status == ToolExecuting && t.done != nil {
			chans = append(chans, t.done)
		}
	}
	e.mu.Unlock()
	if len(chans) == 0 {
		select {
		case <-ctx.Done():
		case <-e.progressSig:
		}
		return
	}
	done := make(chan struct{})
	for _, ch := range chans {
		ch := ch
		go func() {
			<-ch
			select {
			case done <- struct{}{}:
			default:
			}
		}()
	}
	select {
	case <-ctx.Done():
	case <-done:
	case <-e.progressSig:
	}
}

// GetUpdatedContext mirrors getUpdatedContext().
func (e *StreamingToolExecutor) GetUpdatedContext() ToolUseContextPort {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.toolCtx
}

func markToolUseAsComplete(toolUseContext ToolUseContextPort, toolUseID string) {
	toolUseContext.SetInProgressToolUseIDs(func(prev map[string]struct{}) map[string]struct{} {
		next := cloneSet(prev)
		delete(next, toolUseID)
		return next
	})
}

func cloneSet(m map[string]struct{}) map[string]struct{} {
	if m == nil {
		return map[string]struct{}{}
	}
	o := make(map[string]struct{}, len(m))
	for k := range m {
		o[k] = struct{}{}
	}
	return o
}
