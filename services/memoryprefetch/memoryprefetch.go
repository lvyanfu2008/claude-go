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
	"strings"
	"sync"
	"unicode"

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
// surfaced-total byte throttle, collects recent successful tools, and
// starts a background goroutine that calls GetRelevantMemoryAttachments.
// Returns nil when the prefetch should not run (disabled, no input,
// session throttle exceeded).
//
// agents and readFileState may be nil; if nil they default to empty.
func StartRelevantMemoryPrefetch(
	ctx context.Context,
	messages []types.Message,
	cwd string,
	agents []AgentMemoryDef,
	readFileState ReadFileStateChecker,
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
	// Single-word prompts lack enough context for meaningful term extraction.
	// Chinese text doesn't use spaces, so also check CJK character count.
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	if !hasWordBoundary(trimmed) {
		cjkCount := countCJK(trimmed)
		if cjkCount < 4 {
			return nil
		}
	}

	surfacedPaths, surfacedBytes := collectSurfacedMemories(messages)
	if surfacedBytes >= MAX_SESSION_BYTES {
		return nil
	}

	recentTools := collectRecentSuccessfulTools(messages, lastMsg)

	ch := make(chan handleResult, 1)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		memories, err := GetRelevantMemoryAttachments(
			ctx,
			input,
			agents,
			readFileState,
			recentTools,
			surfacedPaths,
			cwd,
		)
		if err != nil || len(memories) == 0 {
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				// Errors in memory prefetch are non-fatal and intentionally
				// never surfaced to the user (mirrors TS .catch(logError)).
				_ = err
			}
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

// GetRelevantMemoryAttachments finds and reads relevant memory files for the
// given user input. It supports agent @-mention routing (agent memory dirs),
// readFileState filtering (dedup files already read via tools), recentTools
// (suppress docs for tools already working), multi-directory concurrent
// search, and surfaced-path de-duplication.
//
// Mirrors TS getRelevantMemoryAttachments in attachments.ts:2197-2243.
func GetRelevantMemoryAttachments(
	ctx context.Context,
	input string,
	agents []AgentMemoryDef,
	readFileState ReadFileStateChecker,
	recentTools []string,
	alreadySurfaced map[string]bool,
	cwd string,
) ([]SurfacedMemory, error) {
	// If an agent is @-mentioned, search only its memory dir (isolation).
	// Otherwise search the auto-memory dir.
	var dirs []string
	mentions := extractAgentMentions(input)
	for _, mention := range mentions {
		agentType := strings.TrimPrefix(mention, "agent-")
		for _, def := range agents {
			if def.AgentType == agentType && def.Memory != "" {
				dirs = append(dirs, memdir.GetAgentMemoryDir(agentType, def.Memory))
			}
		}
	}
	if len(dirs) == 0 {
		dirs = []string{memdir.GetAutoMemPath(cwd)}
	} else {
		// Deduplicate dirs (multiple mentions of same agent)
		seen := make(map[string]bool)
		var unique []string
		for _, d := range dirs {
			if !seen[d] {
				seen[d] = true
				unique = append(unique, d)
			}
		}
		dirs = unique
	}

	// Concurrent search across all memory dirs
	type dirResult struct {
		results []memdir.RelevantMemory
	}
	resultsCh := make(chan dirResult, len(dirs))
	for _, dir := range dirs {
		go func(d string) {
			selected := memdir.FindRelevantMemories(ctx, input, d, recentTools, alreadySurfaced)
			resultsCh <- dirResult{results: selected}
		}(dir)
	}

	var allResults []memdir.RelevantMemory
	for range dirs {
		r := <-resultsCh
		allResults = append(allResults, r.results...)
	}

	// Filter: exclude paths already in readFileState (read by tools this turn)
	// or already surfaced in a prior turn's memory attachment.
	var selected []RelevantMemory
	for _, m := range allResults {
		if readFileState != nil && readFileState.Has(m.Path) {
			continue
		}
		if alreadySurfaced[m.Path] {
			continue
		}
		selected = append(selected, RelevantMemory{Path: m.Path, MtimeMs: m.MtimeMs})
	}
	if len(selected) > 5 {
		selected = selected[:5]
	}

	memories := readMemoriesForSurfacing(selected)
	return memories, nil
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
// Mirrors TS: /\s/.test(trimmed).
func hasWordBoundary(input string) bool {
	for _, r := range input {
		if r == ' ' || r == '\t' || r == '\n' {
			return true
		}
	}
	return false
}

// countCJK returns the number of CJK characters (Han, Katakana, Hiragana)
// in the input. Mirrors TS:
//
//	(trimmed.match(/[\p{Script=Han}\p{Script=Katakana}\p{Script=Hiragana}]/gu) || []).length
func countCJK(input string) int {
	count := 0
	for _, r := range input {
		if unicode.Is(unicode.Han, r) ||
			unicode.Is(unicode.Katakana, r) ||
			unicode.Is(unicode.Hiragana, r) {
			count++
		}
	}
	return count
}
