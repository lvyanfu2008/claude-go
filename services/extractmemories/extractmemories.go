// Package extractmemories mirrors src/services/extractMemories/extractMemories.ts.
//
// After each query-loop turn, extractmemories runs a lightweight sub-agent that
// reviews the recent conversation messages and writes/updates memory files in the
// project's auto-memory directory.  It tracks a cursor (lastMemoryMessageUUID) so
// each invocation only processes messages newer than the previous extraction run.
//
// The extraction sub-agent has restricted tool access:
//   - Read, Glob, Grep — always allowed
//   - Bash — read-only (GOU_DEMO_READONLY_BASH)
//   - Write, Edit — allowed only for paths inside the memory directory
//   - All other tools — denied
package extractmemories

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"goc/conversation-runtime/query"
	"goc/growthbook"
	"goc/memdir"
	"goc/querycontext"
	"goc/tools/localtools"
	"goc/tools/toolexecution"
	"goc/types"
)

// State tracks extraction progress across query turns.
// Mirrors the mutable closure in TS initExtractMemories().
type State struct {
	mu sync.Mutex

	// LastMemoryMessageUUID is the UUID of the last message we processed for extraction.
	// Only messages after this UUID are considered on the next run.
	LastMemoryMessageUUID string

	// TurnsSinceLastExtraction counts consecutive Execute calls where extraction
	// was skipped or throttled. Reset to 0 when extraction actually runs.
	// Mirrors TS turnsSinceLastExtraction counter.
	TurnsSinceLastExtraction int

	// inProgress guards against overlapping extraction runs.
	// When true, new Execute calls stash their params in pendingParams
	// instead of starting a concurrent extraction (coalescing).
	// Mirrors TS inProgress flag in initExtractMemories().
	inProgress bool

	// pendingParams holds the most recent ExtractionParams stashed during
	// an in-progress extraction. Run as a trailing extraction when the
	// current extraction completes. Overwrite semantics — only the latest
	// call is kept. Mirrors TS pendingContext in initExtractMemories().
	pendingParams *ExtractionParams

	// inFlightExtractions tracks completion channels for async extraction
	// goroutines. Each entry is closed when the goroutine completes.
	// Mirrors TS inFlightExtractions Set<Promise<void>>.
	inFlightExtractions []chan struct{}
}

// ExtractionParams carries all context needed to run a single extraction pass.
type ExtractionParams struct {
	// All conversation messages from the current session (pre-query + post-query).
	Messages []types.Message
	// ToolUseContext supplies model config, tools, agent ID, etc.
	ToolUseContext types.ToolUseContext
	// SystemPrompt for the sub-agent (inherited from parent).
	SystemPrompt query.SystemPrompt
	// UserContext key-value pairs.
	UserContext map[string]string
	// SystemContext key-value pairs.
	SystemContext map[string]string
	// Cwd is the working directory for path resolution.
	Cwd string
	// QuerySource label for the sub-agent query.
	QuerySource types.QuerySource
	// NewUUID generates UUIDs (defaults to query.randomUUID).
	NewUUID func() string

	// SkipIndex controls whether MEMORY.md is included in the extraction prompt.
	// Mirrors TS tengu_moth_copse feature flag. When true, the MEMORY.md index
	// is excluded from the sub-agent prompt to reduce noise.
	SkipIndex bool

	// AppendSystemMessage is called when extraction produces memory files.
	// The callback appends a system message (subtype "memory_saved") to the
	// conversation, carrying the list of written memory file paths.
	// If nil, the system message is silently dropped.
	AppendSystemMessage func(msg types.Message)
}

const (
	// entrypointName matches TS memdir ENTRYPOINT_NAME.
	entrypointName = "MEMORY.md"

	// maxExtractionTurns limits the sub-agent's turn count.
	maxExtractionTurns = 5
)

// NewState creates an initialised extraction State.
func NewState() *State {
	return &State{}
}

// extractMemoriesPassportOrHost returns true if post-turn extract-memories should run:
//   - GrowthBook tengu_passport_quail, or
//   - GOC_EXTRACT_MEMORIES=1|true|yes|on (host integration when GB is off).
//
// Set GOC_EXTRACT_MEMORIES=0|false|off|no to disable when using passport_quail in GB.
func extractMemoriesPassportOrHost() bool {
	if growthbook.IsTenguPassportQuail() {
		return true
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOC_EXTRACT_MEMORIES")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return true
	}
	return false
}

// Execute starts an async extraction pass: reviews recent messages and
// runs a restricted sub-agent that may write/update memory files.
// Returns immediately — extraction work runs in a goroutine.
// Results (if any) are delivered via p.AppendSystemMessage callback.
//
// Mirrors TS executeExtractMemories (fire-and-forget, tracked in
// inFlightExtractions, drained at shutdown by DrainPendingExtraction).
func Execute(ctx context.Context, state *State, p ExtractionParams) {
	cwdForLog := strings.TrimSpace(p.Cwd)
	if cwdForLog == "" {
		cwdForLog = "."
	}
	fileExtractMemoriesLogf("enter cwd=%q total_messages=%d last_cursor=%q relax_threshold=%v",
		cwdForLog, len(p.Messages), state.LastMemoryMessageUUID, ExtractMemoriesRelaxThreshold())

	// Guard: auto memory must be enabled.
	if !memdir.IsAutoMemoryEnabled() {
		fileExtractMemoriesLogf("skip reason=auto_memory_disabled")
		return
	}

	// Guard: skip if running inside a sub-agent (only the top-level conversation extracts).
	if p.ToolUseContext.AgentID != nil && strings.TrimSpace(*p.ToolUseContext.AgentID) != "" {
		aid := strings.TrimSpace(*p.ToolUseContext.AgentID)
		if len(aid) > 64 {
			aid = aid[:64] + "…"
		}
		fileExtractMemoriesLogf("skip reason=subagent agent_id=%q", aid)
		return
	}

	// Guard: feature flag gate (TS tengu_passport_quail, default false) or host
	// opt-in (GOC_EXTRACT_MEMORIES=1) for environments without GrowthBook.
	if !extractMemoriesPassportOrHost() {
		fileExtractMemoriesLogf("skip reason=passport_or_host extract_memories_flag=false")
		return
	}

	// Guard: skip if --simple / bare mode (truthy only — CLAUDE_CODE_SIMPLE=0 is off,
	// same as claudemd.IsAutoMemoryEnabled and querycontext.BareModeFromEnv).
	if querycontext.IsEnvTruthy(os.Getenv("CLAUDE_CODE_SIMPLE")) {
		fileExtractMemoriesLogf("skip reason=simple_mode CLAUDE_CODE_SIMPLE_truthy")
		return
	}

	// Guard: isExtractModeActive mirrors TS isExtractModeActive() in memdir/paths.ts.
	// Extraction only runs in interactive sessions by default; non-interactive (-p/--print)
	// sessions are gated behind tengu_slate_thimble.
	if p.ToolUseContext.Options.IsNonInteractiveSession && !growthbook.IsTenguSlateThimble() {
		fileExtractMemoriesLogf("skip reason=non_interactive_slate_thimble_off")
		return
	}

	// Coalescing gate: if an extraction is already in progress, stash
	// params for a trailing run instead of starting a concurrent one.
	// Mirrors TS inProgress check in executeExtractMemoriesImpl.
	state.mu.Lock()
	if state.inProgress {
		state.pendingParams = &p
		state.mu.Unlock()
		fileExtractMemoriesLogf("coalesced reason=extraction_in_progress")
		return
	}
	state.inProgress = true
	state.mu.Unlock()

	// Track in-flight extraction (mirrors TS inFlightExtractions.add(p)).
	done := make(chan struct{})
	state.mu.Lock()
	state.inFlightExtractions = append(state.inFlightExtractions, done)
	state.mu.Unlock()

	go func() {
		defer close(done)

		// Remove from tracking when done (mirrors TS inFlightExtractions.delete(p)).
		defer func() {
			state.mu.Lock()
			for i, ch := range state.inFlightExtractions {
				if ch == done {
					state.inFlightExtractions = append(state.inFlightExtractions[:i], state.inFlightExtractions[i+1:]...)
					break
				}
			}
			state.mu.Unlock()
		}()

		// When the current extraction finishes, dispatch any stashed trailing
		// extraction. Mirrors TS finally block in runExtraction.
		defer func() {
			state.mu.Lock()
			state.inProgress = false
			trailing := state.pendingParams
			state.pendingParams = nil
			state.mu.Unlock()
			if trailing != nil {
				fileExtractMemoriesLogf("trailing extraction start")
				runExtractionCore(ctx, state, *trailing, true)
				fileExtractMemoriesLogf("trailing extraction done")
			}
		}()

		runExtractionCore(ctx, state, p, false)
	}()
}

// newMessagesSinceCursor returns messages newer than the cursor UUID.
// If cursor is empty, returns the last 10 messages (first-run heuristic matching TS).
func newMessagesSinceCursor(messages []types.Message, cursorUUID string) []types.Message {
	if cursorUUID == "" {
		// First run: take the last 50 messages.
		if len(messages) <= 50 {
			return messages
		}
		out := make([]types.Message, 50)
		copy(out, messages[len(messages)-50:])
		return out
	}

	// Find cursor and return everything after it.
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].UUID == cursorUUID {
			return messages[i+1:]
		}
	}
	// Cursor not found: treat all as new.
	return messages
}

// hasMemoryWritesSince checks whether any assistant message after cursorUUID
// contains tool_use blocks for "Write" or "Edit" whose target paths lie
// inside the auto-memory directory.  When the main agent already wrote
// memories, we skip extraction (TS hasMemoryWritesSince).
func hasMemoryWritesSince(messages []types.Message, cursorUUID string) bool {
	all := messages
	if cursorUUID != "" {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].UUID == cursorUUID {
				all = messages[i+1:]
				break
			}
		}
	}

	for _, m := range all {
		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content []struct {
				Type  string          `json:"type"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		for _, block := range payload.Content {
			if block.Type != "tool_use" {
				continue
			}
			if block.Name != "Write" && block.Name != "Edit" {
				continue
			}

			// Extract the file path from the tool input.
			var input struct {
				FilePath string `json:"file_path"`
			}
			if err := json.Unmarshal(block.Input, &input); err != nil {
				continue
			}
			if input.FilePath == "" {
				continue
			}
			if memdir.IsAutoMemPath(input.FilePath) {
				return true
			}
		}
	}
	return false
}

// filterMemoryPaths filters writtenPaths to only those within memoryDir,
// excluding entrypointName (MEMORY.md).
func filterMemoryPaths(writtenPaths []string, memoryDir string) []string {
	cleanMem := strings.TrimSuffix(filepath.Clean(memoryDir), string(filepath.Separator))
	var out []string
	for _, p := range writtenPaths {
		cp := filepath.Clean(p)
		rel, err := filepath.Rel(cleanMem, cp)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if strings.EqualFold(filepath.Base(p), entrypointName) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// extractWrittenPaths scans assistant messages for Write/Edit tool_use blocks
// and returns the file_path values.  Mirrors TS extractWrittenPaths.
func extractWrittenPaths(messages []types.Message) []string {
	var paths []string
	seen := map[string]bool{}
	for _, m := range messages {
		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content []struct {
				Type  string          `json:"type"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		for _, block := range payload.Content {
			if block.Type != "tool_use" {
				continue
			}
			if block.Name != "Write" && block.Name != "Edit" {
				continue
			}
			var input struct {
				FilePath string `json:"file_path"`
			}
			if err := json.Unmarshal(block.Input, &input); err != nil {
				continue
			}
			if input.FilePath == "" || seen[input.FilePath] {
				continue
			}
			seen[input.FilePath] = true
			paths = append(paths, input.FilePath)
		}
	}
	return paths
}

// assistantToolUseSummary counts tool_use `name` values in assistant messages (for
// file logging when no Write/Edit paths were collected).
func assistantToolUseSummary(messages []types.Message) string {
	counts := map[string]int{}
	for _, m := range messages {
		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content []struct {
				Type string `json:"type"`
				Name string `json:"name,omitempty"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		for _, block := range payload.Content {
			if block.Type != "tool_use" || block.Name == "" {
				continue
			}
			counts[block.Name]++
		}
	}
	if len(counts) == 0 {
		return "tool_uses=none"
	}
	var parts []string
	for name, n := range counts {
		parts = append(parts, fmt.Sprintf("%s×%d", name, n))
	}
	sort.Strings(parts)
	return "tool_uses=" + strings.Join(parts, " ")
}

// extractionInitialMessages returns forkContextMessages + extraction user message.
// Mirrors TS runForkedAgent initialMessages = [...forkContextMessages, ...promptMessages].
func extractionInitialMessages(parent []types.Message, extractionUser types.Message) []types.Message {
	return append(slices.Clone(parent), extractionUser)
}

// runExtractionSubagent runs a lightweight sub-agent via query.Query() with
// restricted tool access.  It builds the extraction prompt from newMessages
// and returns the assistant messages produced by the sub-agent.
//
// Message order matches TS [runForkedAgent]: initialMessages = forkContextMessages + promptMessages
// (claude-code-best src/utils/forkedAgent.ts), so the model sees the parent conversation
// before the extraction user turn — "messages above" in the extract prompt.
func runExtractionSubagent(ctx context.Context, p ExtractionParams, memoryDir string, newMessages []types.Message) ([]string, error) {
	subStart := time.Now()
	fileExtractMemoriesLogf("subagent query start fork_messages=%d new_messages_in_prompt=%d max_turns=%d",
		len(p.Messages), len(newMessages), maxExtractionTurns)
	defer func() {
		fileExtractMemoriesLogf("subagent query end duration_ms=%d", time.Since(subStart).Milliseconds())
	}()

	// Build the extraction prompt.
	prompt := buildExtractionPrompt(p, newMessages, memoryDir)

	newUUID := p.NewUUID
	if newUUID == nil {
		newUUID = query.RandomUUID
	}
	userMsg := buildExtractionUserMessage(prompt, newUUID)
	msgs := extractionInitialMessages(p.Messages, userMsg)

	// Clone the tool context for the sub-agent.
	tc := p.ToolUseContext
	if tc.AgentID != nil {
		// Clear agent ID so extraction runs as top-level (matters for guards).
		tc.AgentID = nil
	}
	tc.Options.IsNonInteractiveSession = true

	// Limit turns for the extraction agent.
	maxTurns := maxExtractionTurns

	// Build tool execution deps with restricted InvokeTool.
	qdeps := query.ProductionDeps(nil, nil)
	qdeps.NewUUID = newUUID
	qdeps.ToolexecutionDeps = buildRestrictedExecutionDeps(memoryDir)

	qp := query.QueryParams{
		Messages:        msgs,
		SystemPrompt:    p.SystemPrompt,
		UserContext:     p.UserContext,
		SystemContext:   p.SystemContext,
		ToolUseContext:  tc,
		QuerySource:     types.QuerySource("extract_memories"),
		StreamingParity: true,
		MaxTurns:        &maxTurns,
		Deps:            &qdeps,
	}

	// Use a fresh context so the extraction sub-agent isn't canceled
	// when the parent query's context times out or is canceled.
	subCtx := context.Background()

	// Collect all yielded messages from the sub-agent query.
	var assistantMessages []types.Message
	for y, err := range query.Query(subCtx, qp) {
		if err != nil {
			return nil, err
		}
		if y.Message != nil && y.Message.Type == types.MessageTypeAssistant {
			assistantMessages = append(assistantMessages, *y.Message)
		}
	}

	written := extractWrittenPaths(assistantMessages)
	if len(written) == 0 {
		fileExtractMemoriesLogf("subagent no_write_or_edit_paths assistant_messages=%d %s",
			len(assistantMessages), assistantToolUseSummary(assistantMessages))
	}
	return written, nil
}

// runExtractionCore is the inner extraction logic: throttle, new-message
// detection, sub-agent invocation, cursor update, and result reporting.
// When isTrailingRun is true, the throttle check is skipped (mirrors TS
// isTrailingRun parameter in executeExtractMemoriesImpl).
func runExtractionCore(ctx context.Context, state *State, p ExtractionParams, isTrailingRun bool) ([]string, error) {
	cwd := strings.TrimSpace(p.Cwd)
	if cwd == "" {
		cwd = "."
	}
	memoryDir := memdir.GetAutoMemPath(cwd)
	if memoryDir == "" {
		fileExtractMemoriesLogf("skip reason=empty_memory_dir cwd=%q", cwd)
		return nil, nil
	}
	fileExtractMemoriesLogf("memory_dir=%q", memoryDir)

	_ = memdir.EnsureMemoryDirExists(memoryDir)

	if !isTrailingRun {
		throttle := growthbook.GetTenguBrambleLintel()
		state.mu.Lock()
		state.TurnsSinceLastExtraction++
		turnCount := state.TurnsSinceLastExtraction
		state.mu.Unlock()
		if turnCount < throttle {
			fileExtractMemoriesLogf("skip reason=throttle turn_since_extraction=%d throttle=%d", turnCount, throttle)
			return nil, nil
		}
	}

	// Determine which messages are "new" since the last extraction.
	newMessages := newMessagesSinceCursor(p.Messages, state.LastMemoryMessageUUID)
	if len(newMessages) == 0 {
		fileExtractMemoriesLogf("skip reason=no_new_messages")
		return nil, nil
	}
	fileExtractMemoriesLogf("new_messages=%d", len(newMessages))

	// Check if the main agent already wrote memory files in this turn.
	if hasMemoryWritesSince(p.Messages, state.LastMemoryMessageUUID) {
		fileExtractMemoriesLogf("skip reason=main_agent_already_wrote_memory")
		if len(p.Messages) > 0 {
			last := p.Messages[len(p.Messages)-1]
			state.mu.Lock()
			state.LastMemoryMessageUUID = last.UUID
			state.mu.Unlock()
		}
		return nil, nil
	}

	// Build the extraction prompt and run the sub-agent.
	writtenPaths, err := runExtractionSubagent(ctx, p, memoryDir, newMessages)
	if err != nil {
		fileExtractMemoriesLogf("error subagent: %v", err)
		return nil, fmt.Errorf("extractmemories: %w", err)
	}
	fileExtractMemoriesLogf("subagent raw_written_paths=%d paths=%q", len(writtenPaths), writtenPaths)

	// Filter out MEMORY.md itself.
	memoryPaths := filterMemoryPaths(writtenPaths, memoryDir)
	fileExtractMemoriesLogf("after_filter memory_paths=%d paths=%q", len(memoryPaths), memoryPaths)

	// Update cursor: use the UUID of the last message in the full list.
	if len(p.Messages) > 0 {
		last := p.Messages[len(p.Messages)-1]
		state.mu.Lock()
		state.LastMemoryMessageUUID = last.UUID
		state.TurnsSinceLastExtraction = 0
		state.mu.Unlock()
	}

	if len(memoryPaths) == 0 {
		fileExtractMemoriesLogf("done reason=no_files_after_filter (no memory_saved message)")
		return nil, nil
	}

	// Emit "memory saved" system message (TS createMemorySavedMessage).
	if p.AppendSystemMessage != nil {
		newUUID := p.NewUUID
		if newUUID == nil {
			newUUID = query.RandomUUID
		}
		msg := createMemorySavedMessage(memoryPaths, newUUID)
		p.AppendSystemMessage(msg)
		fileExtractMemoriesLogf("done memory_saved paths=%q append_callback=true", memoryPaths)
	} else {
		fileExtractMemoriesLogf("done memory_paths=%d append_callback=nil (no memory_saved message)", len(memoryPaths))
	}

	return memoryPaths, nil
}

// DrainPendingExtraction waits for any in-flight extraction on state to complete.
// Mirrors TS drainPendingExtraction in extractMemories.ts.
//
// Default timeout is 60s (matches TS PENDING_EXTRACTION_TIMEOUT_MS).
// The timeout is a soft cap — once hit, DrainPendingExtraction returns
// without waiting for remaining extractions. Called at shutdown after
// response flush to let the forked agent complete before the 5s
// gracefulShutdownSync failsafe kills it.
func DrainPendingExtraction(state *State, timeoutMs ...int) {
	timeout := 60000 // 60s default, matches TS
	if len(timeoutMs) > 0 && timeoutMs[0] > 0 {
		timeout = timeoutMs[0]
	}

	// Snapshot in-flight channels so we don't hold the lock while waiting.
	state.mu.Lock()
	if len(state.inFlightExtractions) == 0 {
		state.mu.Unlock()
		return
	}
	channels := make([]chan struct{}, len(state.inFlightExtractions))
	copy(channels, state.inFlightExtractions)
	state.mu.Unlock()

	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	for _, ch := range channels {
		select {
		case <-ch:
			// This extraction completed.
		case <-timer.C:
			// Soft timeout — return without waiting for the rest.
			return
		}
	}
}

// buildExtractionUserMessage creates a user message with the extraction prompt.
// The message is marked IsMeta=true (system-generated, not user input) so
// startRelevantMemoryPrefetch and isHumanTurn skip it.
func buildExtractionUserMessage(prompt string, newUUID func() string) types.Message {
	uuid := newUUID()
	content := map[string]any{
		"role":    "user",
		"content": prompt,
	}
	b, _ := json.Marshal(content)
	isMeta := true
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    uuid,
		Message: b,
		Content: func() json.RawMessage { b, _ := json.Marshal(prompt); return b }(),
		IsMeta:  &isMeta,
	}
}

// buildRestrictedExecutionDeps returns ToolexecutionDeps that restrict tools
// for the extraction sub-agent.
func buildRestrictedExecutionDeps(memoryDir string) toolexecution.ExecutionDeps {
	memDir := strings.TrimSpace(memoryDir)
	invokeReadFileState := localtools.NewReadFileState()

	dispatchTool := func(ctx context.Context, name string, input json.RawMessage) (string, bool, error) {
		switch name {
		case "Read":
			return localtools.ReadFromJSON(input, nil, invokeReadFileState, nil)
		case "Glob":
			return localtools.GlobFromJSON(ctx, input, nil)
		case "Grep":
			return localtools.GrepFromJSON(ctx, input, nil)
		case "Bash":
			return localtools.BashFromJSON(ctx, input, "", true, "")
		case "Write":
			if memDir != "" {
				if fp := filePathFromInput(input); fp == "" || !memdir.IsAutoMemPath(fp) {
					return "", false, fmt.Errorf("Write: path not in memory directory")
				}
			}
			return localtools.WriteFromJSONDeps(input, nil, invokeReadFileState, nil)
		case "Edit":
			if memDir != "" {
				if fp := filePathFromInput(input); fp == "" || !memdir.IsAutoMemPath(fp) {
					return "", false, fmt.Errorf("Edit: path not in memory directory")
				}
			}
			return localtools.EditFromJSONDeps(input, nil, invokeReadFileState, false, nil)
		default:
			return "", false, fmt.Errorf("tool %q not allowed in extraction sub-agent", name)
		}
	}

	return toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, _ string, input json.RawMessage) (string, bool, error) {
			// Allow REPL — when REPL mode is enabled (ant-default), primitive
			// tools are hidden from the outer tool list so the forked agent
			// calls REPL instead. Each inner primitive is re-dispatched
			// through dispatchTool, so the Read/Bash/Edit/Write gates below
			// still apply. Mirrors TS extractMemories.ts:180-182.
			if name == "REPL" {
				return dispatchREPLTool(ctx, input, dispatchTool)
			}
			return dispatchTool(ctx, name, input)
		},
	}
}

// replToolInput matches the TS REPL tool input shape (src/tools/REPLTool/).
type replToolInput struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
	Batch []struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"batch"`
}

// dispatchREPLTool parses a REPL tool input and dispatches each inner tool
// call through dispatchTool. Inner tools are re-gated by the same
// Read/Glob/Grep/Bash/Write/Edit restrictions.
func dispatchREPLTool(ctx context.Context, input json.RawMessage, dispatch func(ctx context.Context, name string, input json.RawMessage) (string, bool, error)) (string, bool, error) {
	var in replToolInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", true, err
	}

	type step struct {
		name  string
		input json.RawMessage
	}
	var steps []step
	if len(in.Batch) > 0 {
		for _, b := range in.Batch {
			raw := b.Input
			if raw == nil {
				raw = json.RawMessage(`{}`)
			}
			steps = append(steps, step{name: b.Name, input: raw})
		}
	} else {
		raw := in.Input
		if raw == nil {
			raw = json.RawMessage(`{}`)
		}
		steps = append(steps, step{name: in.Tool, input: raw})
	}

	if len(steps) == 0 {
		return "", true, fmt.Errorf("REPL input: use {\"tool\":\"Read\",\"input\":{...}} or {\"batch\":[{\"name\":\"Read\",\"input\":{...}}]}")
	}

	var blocks []string
	for i, st := range steps {
		nm := strings.TrimSpace(st.name)
		if nm == "" || nm == "REPL" {
			return "", true, fmt.Errorf("REPL step %d: invalid tool name %q", i, st.name)
		}
		out, isErr, err := dispatch(ctx, nm, st.input)
		if err != nil {
			return "", true, err
		}
		prefix := fmt.Sprintf("[%s] ", nm)
		if isErr {
			prefix = fmt.Sprintf("[%s ERROR] ", nm)
		}
		blocks = append(blocks, prefix+strings.TrimSpace(out))
	}

	return strings.Join(blocks, "\n\n"), false, nil
}

// filePathFromInput extracts the file_path from a Write/Edit tool input.
func filePathFromInput(input json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(input, &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v.FilePath)
}

// countModelVisibleMessages returns the number of messages that are visible
// to the model (user + assistant types).  System, progress, and other
// non-model-visible message types are excluded.
func countModelVisibleMessages(msgs []types.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Type == types.MessageTypeUser || m.Type == types.MessageTypeAssistant {
			n++
		}
	}
	return n
}

// createMemorySavedMessage creates a system message with subtype "memory_saved"
// carrying the list of written memory file paths.
// Mirrors TS createMemorySavedMessage in src/utils/messages.ts.
func createMemorySavedMessage(writtenPaths []string, newUUID func() string) types.Message {
	uuid := newUUID()
	now := time.Now().Format(time.RFC3339)
	isMeta := false
	subtype := types.SubtypeMemorySaved
	return types.Message{
		Type:         types.MessageTypeSystem,
		UUID:         uuid,
		Subtype:      &subtype,
		WrittenPaths: writtenPaths,
		Timestamp:    &now,
		IsMeta:       &isMeta,
	}
}
