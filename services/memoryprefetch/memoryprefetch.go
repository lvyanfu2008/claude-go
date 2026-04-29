// Package memoryprefetch mirrors the TS relevant-memory prefetch pattern
// (src/utils/attachments.ts startRelevantMemoryPrefetch / MemoryPrefetch).
//
// A Handle is created once per user turn in queryLoop before the model loop
// starts. It runs FindRelevantMemories + file reads in a background goroutine
// while the model streams and tools execute. After tool results, the streaming
// loop calls Poll() (non-blocking) to consume results if ready; otherwise it
// skips and retries next iteration.
package memoryprefetch

import (
	"context"
	"errors"
	"sync"

	"goc/growthbook"
	"goc/memdir"
	"goc/types"
)

// Handle mirrors TS MemoryPrefetch. It wraps an async selector + read
// operation so the main model loop never blocks on memory selection.
type Handle struct {
	mu   sync.Mutex
	ch   <-chan handleResult
	done chan struct{}

	// settled is true once the background goroutine has completed.
	settled bool
	// consumed is true after the first Poll() call that returns results.
	consumed bool
}

type handleResult struct {
	memories []SurfacedMemory
	err      error
}

// StartRelevantMemoryPrefetch mirrors TS startRelevantMemoryPrefetch.
// It checks feature flags, extracts the last user message, scans for
// surfaced-total byte throttle, and starts a background goroutine.
// Returns nil when the prefetch should not run (disabled, no input,
// session throttle exceeded).
func StartRelevantMemoryPrefetch(
	ctx context.Context,
	messages []types.Message,
	cwd string,
) *Handle {
	if !memdir.IsAutoMemoryEnabled() {
		return nil
	}
	if !growthbook.IsTenguMothCopse() {
		return nil
	}

	lastMsg, ok := getLastUserMessage(messages)
	if !ok {
		return nil
	}
	input := getUserMessageText(lastMsg)
	if input == "" || !hasWordBoundary(input) {
		return nil
	}

	surfacedPaths, surfacedBytes := collectSurfacedMemories(messages)
	if surfacedBytes >= MAX_SESSION_BYTES {
		return nil
	}

	memoryDir := memdir.GetAutoMemPath(cwd)

	ch := make(chan handleResult, 1)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		// Find relevant memory paths using Sonnet side-query
		selected := memdir.FindRelevantMemories(
			ctx,
			input,
			memoryDir,
			nil, // recentTools — not yet wired
			surfacedPaths,
		)
		if len(selected) == 0 {
			return
		}
		// Convert to RelevantMemory
		var sel []RelevantMemory
		for _, s := range selected {
			sel = append(sel, RelevantMemory{
				Path:    s.Path,
				MtimeMs: s.MtimeMs,
			})
		}
		memories := readMemoriesForSurfacing(sel)
		if len(memories) == 0 {
			return
		}
		// Notify result; Poll() reads this
		select {
		case ch <- handleResult{memories: memories}:
		case <-done:
		}
	}()

	return &Handle{
		ch:   ch,
		done: done,
	}
}

// Poll returns the memories if the background goroutine has finished and
// the results haven't been consumed yet. Returns nil, nil otherwise.
// Non-blocking — the caller continues the main loop immediately.
func (h *Handle) Poll() []types.Message {
	if h == nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.consumed || h.settled {
		return nil
	}

	// Non-blocking receive from channel
	select {
	case res, ok := <-h.ch:
		if !ok {
			// Channel closed without result — treat as error/settled
			h.settled = true
			return nil
		}
		h.settled = true
		if res.err != nil || len(res.memories) == 0 {
			return nil
		}
		h.consumed = true
		// Truncate to 5 — matching TS max cap
		mems := res.memories
		if len(mems) > 5 {
			mems = mems[:5]
		}
		msg := newRelevantMemoriesAttachment(mems)
		return []types.Message{msg}
	default:
		// Not ready yet — skip and retry next iteration
		return nil
	}
}

// Close aborts the in-flight request and stops the background goroutine.
// Mirrors TS MemoryPrefetch.[Symbol.dispose]().
func (h *Handle) Close() {
	if h == nil {
		return
	}
	select {
	case <-h.done:
		// Already closed
	default:
		close(h.done)
	}
}

// ErrNoResult is returned when the selector found no relevant memories.
var ErrNoResult = errors.New("memoryprefetch: no relevant memories found")

// hasWordBoundary returns true if the input contains at least one whitespace
// character, ensuring the query has enough context for meaningful selection.
// Mirrors TS: !input || !/\s/.test(input.trim()) → skip single-word prompts.
func hasWordBoundary(input string) bool {
	for _, r := range input {
		if r == ' ' || r == '\t' || r == '\n' {
			return true
		}
	}
	return false
}
