package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"

	"goc/ccb-engine/internal/protocol"
)

// BridgeRunner implements ToolRunner by emitting execute_tool on the wire and waiting for a ToolResult line.
type BridgeRunner struct {
	ctx context.Context

	writeMu *sync.Mutex
	write   func(protocol.StreamEvent) error

	mu               sync.Mutex
	waiters          map[string]chan bridgeToolResult
	stateRevProvider func() uint64
	// toolExecutePolicy is attached to each execute_tool event (optional; e.g. decision hint for TS).
	toolExecutePolicy map[string]any
}

type bridgeToolResult struct {
	content string
	isError bool
}

// NewBridgeRunner returns a runner that writes execute_tool events via write (must be mutex-guarded if shared).
func NewBridgeRunner(ctx context.Context, writeMu *sync.Mutex, write func(protocol.StreamEvent) error) *BridgeRunner {
	return &BridgeRunner{
		ctx:     ctx,
		writeMu: writeMu,
		write:   write,
		waiters: make(map[string]chan bridgeToolResult),
	}
}

// SetStateRevProvider sets a callback for current session stateRev (for execute_tool events).
func (b *BridgeRunner) SetStateRevProvider(fn func() uint64) {
	b.stateRevProvider = fn
}

// SetToolExecutePolicy sets optional policy metadata on every execute_tool (nil clears).
func (b *BridgeRunner) SetToolExecutePolicy(p map[string]any) {
	b.toolExecutePolicy = p
}

// Run implements ToolRunner.
func (b *BridgeRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	callID := randomHexID()
	ch := make(chan bridgeToolResult, 1)
	b.mu.Lock()
	b.waiters[callID] = ch
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.waiters, callID)
		b.mu.Unlock()
	}()

	var inputObj map[string]any
	if len(input) > 0 {
		_ = json.Unmarshal(input, &inputObj)
	}
	if inputObj == nil {
		inputObj = map[string]any{}
	}
	var rev uint64
	if b.stateRevProvider != nil {
		rev = b.stateRevProvider()
	}
	ev := protocol.ExecuteTool(callID, toolUseID, name, inputObj, rev, b.toolExecutePolicy)
	b.writeMu.Lock()
	err := b.write(ev)
	b.writeMu.Unlock()
	if err != nil {
		return "", true, err
	}

	runCtx := b.ctx
	if runCtx == nil {
		runCtx = ctx
	}
	select {
	case res := <-ch:
		return res.content, res.isError, nil
	case <-runCtx.Done():
		return "", true, runCtx.Err()
	case <-ctx.Done():
		return "", true, ctx.Err()
	}
}

// DeliverToolResult matches a client ToolResult line to a pending wait.
func (b *BridgeRunner) DeliverToolResult(callID, toolUseID, content string, isError bool) bool {
	b.mu.Lock()
	ch, ok := b.waiters[callID]
	b.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- bridgeToolResult{content: content, isError: isError}:
		return true
	default:
		return false
	}
}

func randomHexID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "ccb_fallback_id"
	}
	return hex.EncodeToString(b[:])
}
