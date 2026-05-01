// Package sessionmemory provides session memory extraction — a post-turn hook
// that maintains a markdown notes file (summary.md) about the current conversation
// using a forked subagent. Mirrors src/services/SessionMemory/sessionMemory.ts.
package sessionmemory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"goc/compactquerysource"
	"goc/compactservice"
	"goc/conversation-runtime/query"
	"goc/growthbook"
	"goc/memdir"
	"goc/tools/localtools"
	"goc/tools/toolexecution"
	"goc/types"
)

// maxExtractionTurns limits the sub-agent's turn count for session memory.
const maxExtractionTurns = 5

// hasToolCallsInLastAssistantTurn scans messages backwards and returns true
// if the most recent assistant message contains any tool_use content blocks.
// Mirrors TS hasToolCallsInLastAssistantTurn in utils/messages.ts.
func hasToolCallsInLastAssistantTurn(messages []types.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		var blocks []struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payload.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" {
				return true
			}
		}
		return false
	}
	return false
}

// countToolCallsSince counts tool_use blocks in assistant messages after
// the message identified by sinceUUID (exclusive). If sinceUUID is empty,
// counts from the beginning.
// Mirrors TS countToolCallsSince.
func countToolCallsSince(messages []types.Message, sinceUUID string) int {
	toolCallCount := 0
	foundStart := sinceUUID == ""

	for _, m := range messages {
		if !foundStart {
			if m.UUID == sinceUUID {
				foundStart = true
			}
			continue
		}

		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		var blocks []struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payload.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" {
				toolCallCount++
			}
		}
	}

	return toolCallCount
}

// shouldExtractMemory checks thresholds to determine whether a session memory
// extraction should run for the current turn. Mirrors TS shouldExtractMemory.
func shouldExtractMemory(state *State, messages []types.Message) bool {
	currentTokenCount := compactservice.TokenCountWithEstimation(messages)

	if !state.IsSessionMemoryInitialized() {
		if !state.HasMetInitializationThreshold(currentTokenCount) {
			return false
		}
		state.MarkSessionMemoryInitialized()
	}

	hasMetTokenThreshold := state.HasMetUpdateThreshold(currentTokenCount)

	toolCallsSince := countToolCallsSince(messages, state.LastMemoryMessageUUID)
	hasMetToolCallThreshold := toolCallsSince >= state.GetToolCallsBetweenUpdates()

	hasToolCallsInLastTurn := hasToolCallsInLastAssistantTurn(messages)

	shouldExtract := (hasMetTokenThreshold && hasMetToolCallThreshold) ||
		(hasMetTokenThreshold && !hasToolCallsInLastTurn)

	if shouldExtract {
		if last := messages[len(messages)-1]; last.UUID != "" {
			state.LastMemoryMessageUUID = last.UUID
		}
		return true
	}

	return false
}

// initSessionMemoryConfigIfNeeded merges remote GrowthBook config into state
// on first call. Subsequent calls are no-ops. Mirrors TS initSessionMemoryConfigIfNeeded.
func initSessionMemoryConfigIfNeeded(state *State) {
	state.mu.Lock()
	if state.ConfigInitialized {
		state.mu.Unlock()
		return
	}
	state.ConfigInitialized = true
	state.mu.Unlock()

	// Try to load remote config from GrowthBook (non-blocking, may be stale).
	// Falls back to DefaultConfig if no remote config is set.
	_ = growthbook.DefaultManager() // ensure initialized
	remote := growthbook.DefaultManager().Get("tengu_sm_config")

	cfg := DefaultConfig

	if m, ok := remote.(map[string]any); ok {
		if v, ok := m["minimumMessageTokensToInit"].(float64); ok && v > 0 {
			cfg.MinimumMessageTokensToInit = int(v)
		}
		if v, ok := m["minimumTokensBetweenUpdate"].(float64); ok && v > 0 {
			cfg.MinimumTokensBetweenUpdate = int(v)
		}
		if v, ok := m["toolCallsBetweenUpdates"].(float64); ok && v > 0 {
			cfg.ToolCallsBetweenUpdates = int(v)
		}
	}

	state.SetConfig(cfg)
}

// setupSessionMemoryFile creates the session memory directory and file if
// they don't exist (using the template), clears the readFileState cache for
// the file, reads the current content, and returns the path and content.
// Mirrors TS setupSessionMemoryFile.
func setupSessionMemoryFile(sessionID, cwd string, readFileState *localtools.ReadFileState) (memoryPath string, currentMemory string, err error) {
	sessionMemoryDir := memdir.GetSessionMemoryDir(sessionID, cwd)
	if sessionMemoryDir == "" {
		return "", "", fmt.Errorf("sessionmemory: could not determine session memory directory")
	}

	if err := os.MkdirAll(sessionMemoryDir, 0o700); err != nil {
		return "", "", fmt.Errorf("sessionmemory: mkdir %s: %w", sessionMemoryDir, err)
	}

	memoryPath = memdir.GetSessionMemoryPath(sessionID, cwd)

	// Create the file with template if it doesn't exist (O_EXCL).
	f, err := os.OpenFile(memoryPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return "", "", fmt.Errorf("sessionmemory: create %s: %w", memoryPath, err)
		}
		// File already exists — expected path.
	} else {
		// File was just created — write template.
		template := LoadSessionMemoryTemplate()
		if _, err := f.WriteString(template); err != nil {
			f.Close()
			return "", "", fmt.Errorf("sessionmemory: write template %s: %w", memoryPath, err)
		}
		f.Close()
	}

	// Drop any cached entry so the Read below gets real content (not a
	// file_unchanged stub from FileReadTool dedup).
	if readFileState != nil {
		abs, _ := filepath.Abs(memoryPath)
		readFileState.Delete(abs)
	}

	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return "", "", fmt.Errorf("sessionmemory: read %s: %w", memoryPath, err)
	}

	return memoryPath, string(data), nil
}

// runSessionMemorySubagent dispatches a forked subagent via query.Query with
// restricted tool access (only Edit on the exact memoryPath). It returns
// nil on success. Mirrors TS runForkedAgent usage in extractSessionMemory.
func runSessionMemorySubagent(
	ctx context.Context,
	params query.QueryCompleteParams,
	memoryPath string,
	readFileState *localtools.ReadFileState,
) error {
	// Build the update prompt.
	currentMemory, err := os.ReadFile(memoryPath)
	if err != nil {
		return fmt.Errorf("sessionmemory: read %s: %w", memoryPath, err)
	}
	userPrompt := BuildSessionMemoryUpdatePrompt(string(currentMemory), memoryPath)

	// Create user message with extraction prompt (IsMeta=true so it's not
	// treated as a real user turn by downstream hooks).
	userMsg := createMetaUserMessage(userPrompt)

	// Build initial messages: fork context (parent conversation) + extraction user message.
	initialMessages := make([]types.Message, 0, len(params.Messages)+1)
	initialMessages = append(initialMessages, params.Messages...)
	initialMessages = append(initialMessages, userMsg)

	// Clone ToolUseContext for the subagent (clear AgentID so guards treat
	// it as top-level).
	tc := params.ToolUseContext
	tc.AgentID = nil
	tc.Options.IsNonInteractiveSession = true

	maxTurns := maxExtractionTurns
	restrictedDeps := buildSessionMemoryExecutionDeps(memoryPath, readFileState)

	qdeps := query.QueryDeps{
		NewUUID:           query.RandomUUID,
		ToolexecutionDeps: restrictedDeps,
	}

	qp := query.QueryParams{
		Messages:        initialMessages,
		SystemPrompt:    params.SystemPrompt,
		UserContext:     params.UserContext,
		SystemContext:   params.SystemContext,
		ToolUseContext:  tc,
		QuerySource:     types.QuerySource("session_memory"),
		StreamingParity: true,
		MaxTurns:        &maxTurns,
		Deps:            &qdeps,
	}

	// Consume all yields from the subagent query (ignore errors from
	// individual yields — the subagent just needs to run).
	for _, err := range query.Query(ctx, qp) {
		if err != nil {
			return fmt.Errorf("sessionmemory: subagent query: %w", err)
		}
	}

	return nil
}

// buildSessionMemoryExecutionDeps returns ExecutionDeps that restrict tools
// for the session memory subagent: only Edit on the exact memoryPath is allowed.
func buildSessionMemoryExecutionDeps(memoryPath string, readFileState *localtools.ReadFileState) toolexecution.ExecutionDeps {
	canUseTool := CreateMemoryFileCanUseTool(memoryPath)

	return toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
			// Check permission first.
			decision, err := canUseTool(ctx, name, toolUseID, input)
			if err != nil {
				return "", true, err
			}
			if decision.Behavior != toolexecution.PermissionAllow {
				return "", true, fmt.Errorf("tool %q not allowed in session memory subagent", name)
			}

			// Only Edit is allowed — dispatches through localtools.
			if name == fileEditToolName {
				return localtools.EditFromJSONDeps(input, nil, readFileState, false, nil)
			}

			return "", true, fmt.Errorf("tool %q not allowed in session memory subagent", name)
		},
	}
}

// updateLastSummarizedMessageIdIfSafe updates the last summarized message ID
// if the last message has no tool calls (safe to mark as summarized).
// Mirrors TS updateLastSummarizedMessageIdIfSafe.
func updateLastSummarizedMessageIdIfSafe(state *State, messages []types.Message) {
	if !hasToolCallsInLastAssistantTurn(messages) {
		if last := messages[len(messages)-1]; last.UUID != "" {
			state.SetLastSummarizedMessageID(last.UUID)
		}
	}
}

// Hook returns an OnQueryComplete callback that runs session memory extraction
// as a post-turn hook. Only one extraction runs at a time (sequential gate).
// Mirrors TS initSessionMemory/extractSessionMemory.
func Hook(state *State, sessionID, cwd string) func(ctx context.Context, params query.QueryCompleteParams) {
	var mu sync.Mutex

	return func(ctx context.Context, params query.QueryCompleteParams) {
		// Only run on main REPL thread (not subagents, teammates, etc.).
		if !compactquerysource.MainThreadLike(string(params.QuerySource)) {
			return
		}

		// Check gate lazily (cached, non-blocking).
		if !growthbook.IsTenguSessionMemory() {
			state.mu.Lock()
			if !state.HasLoggedGateFailure {
				state.HasLoggedGateFailure = true
				state.mu.Unlock()
				// TODO: logEvent('tengu_session_memory_gate_disabled', {})
			} else {
				state.mu.Unlock()
			}
			return
		}

		// Initialize config from remote (lazy, only once).
		initSessionMemoryConfigIfNeeded(state)

		if !shouldExtractMemory(state, params.Messages) {
			return
		}

		// Run extraction in a goroutine; sequential gate prevents overlapping runs.
		go func() {
			mu.Lock()
			defer mu.Unlock()

			state.MarkExtractionStarted()
			defer state.MarkExtractionCompleted()

			extractOnce(ctx, state, sessionID, cwd, params)
		}()
	}
}

// extractOnce runs a single session memory extraction pass.
func extractOnce(
	ctx context.Context,
	state *State,
	sessionID, cwd string,
	params query.QueryCompleteParams,
) {
	// Create isolated ReadFileState so the subagent doesn't pollute the parent cache.
	setupRFS := localtools.NewReadFileState()

	memoryPath, _, err := setupSessionMemoryFile(sessionID, cwd, setupRFS)
	if err != nil {
		// Non-fatal: if we can't set up the file, log and skip.
		// TODO: logEvent
		return
	}

	if err := runSessionMemorySubagent(ctx, params, memoryPath, setupRFS); err != nil {
		// Non-fatal: subagent errors don't affect the main conversation.
		// TODO: logEvent
		return
	}

	// Log extraction event for tracking frequency.
	// TODO: logEvent('tengu_session_memory_extraction', {...})

	// Record the context size at extraction for minimumTokensBetweenUpdate.
	state.RecordExtractionTokenCount(compactservice.TokenCountWithEstimation(params.Messages))

	// Update lastSummarizedMessageId after successful completion.
	updateLastSummarizedMessageIdIfSafe(state, params.Messages)
}

// ManualExtractionResult mirrors TS ManualExtractionResult.
type ManualExtractionResult struct {
	Success    bool   `json:"success"`
	MemoryPath string `json:"memoryPath,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ManuallyExtract runs session memory extraction bypassing threshold checks.
// Used by the /summary command. Mirrors TS manuallyExtractSessionMemory.
func ManuallyExtract(
	ctx context.Context,
	state *State,
	sessionID, cwd string,
	messages []types.Message,
	toolUseContext types.ToolUseContext,
	systemPrompt query.SystemPrompt,
	userContext, systemContext map[string]string,
) ManualExtractionResult {
	if len(messages) == 0 {
		return ManualExtractionResult{Success: false, Error: "No messages to summarize"}
	}

	state.MarkExtractionStarted()
	defer state.MarkExtractionCompleted()

	// Create isolated ReadFileState.
	setupRFS := localtools.NewReadFileState()

	memoryPath, _, err := setupSessionMemoryFile(sessionID, cwd, setupRFS)
	if err != nil {
		return ManualExtractionResult{Success: false, Error: err.Error()}
	}

	// Build params for the subagent.
	params := query.QueryCompleteParams{
		Messages:       messages,
		SystemPrompt:   systemPrompt,
		UserContext:    userContext,
		SystemContext:  systemContext,
		ToolUseContext: toolUseContext,
		QuerySource:    types.QuerySource("session_memory_manual"),
		Cwd:            cwd,
	}

	if err := runSessionMemorySubagent(ctx, params, memoryPath, setupRFS); err != nil {
		return ManualExtractionResult{Success: false, Error: err.Error()}
	}

	// Log manual extraction event.
	// TODO: logEvent('tengu_session_memory_manual_extraction', {})

	// Record the context size at extraction.
	state.RecordExtractionTokenCount(compactservice.TokenCountWithEstimation(messages))

	// Update lastSummarizedMessageId after successful completion.
	updateLastSummarizedMessageIdIfSafe(state, messages)

	return ManualExtractionResult{Success: true, MemoryPath: memoryPath}
}

// createMetaUserMessage creates a user message with IsMeta=true, mirroring
// TS createUserMessage({ content, isMeta: true }).
func createMetaUserMessage(prompt string) types.Message {
	isMeta := true
	content := map[string]any{
		"role":    "user",
		"content": prompt,
	}
	b, _ := json.Marshal(content)
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    query.RandomUUID(),
		Message: b,
		IsMeta:  &isMeta,
	}
}
