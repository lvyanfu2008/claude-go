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
	"sort"
	"strings"
	"sync"
	"time"

	"goc/claudemd"
	"goc/conversation-runtime/query"
	"goc/growthbook"
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

// Execute runs one extraction pass: reviews recent messages and runs a
// restricted sub-agent that may write/update memory files.
//
// Returns the list of written memory file paths (excluding MEMORY.md).
// Returns nil, nil when extraction is skipped (disabled, inside agent, no new messages).
//
// Mirrors TS executeExtractMemories.
func Execute(ctx context.Context, state *State, p ExtractionParams) ([]string, error) {
	cwdForLog := strings.TrimSpace(p.Cwd)
	if cwdForLog == "" {
		cwdForLog = "."
	}
	fileExtractMemoriesLogf("enter cwd=%q total_messages=%d last_cursor=%q relax_threshold=%v",
		cwdForLog, len(p.Messages), state.LastMemoryMessageUUID, ExtractMemoriesRelaxThreshold())

	// Guard: auto memory must be enabled.
	if !claudemd.IsAutoMemoryEnabled() {
		fileExtractMemoriesLogf("skip reason=auto_memory_disabled")
		return nil, nil
	}

	// Guard: skip if running inside a sub-agent (only the top-level conversation extracts).
	if p.ToolUseContext.AgentID != nil && strings.TrimSpace(*p.ToolUseContext.AgentID) != "" {
		aid := strings.TrimSpace(*p.ToolUseContext.AgentID)
		if len(aid) > 64 {
			aid = aid[:64] + "…"
		}
		fileExtractMemoriesLogf("skip reason=subagent agent_id=%q", aid)
		return nil, nil
	}

	// Guard: feature flag gate (TS tengu_passport_quail, default false) or host
	// opt-in (GOC_EXTRACT_MEMORIES=1) for environments without GrowthBook.
	if !extractMemoriesPassportOrHost() {
		fileExtractMemoriesLogf("skip reason=passport_or_host extract_memories_flag=false")
		return nil, nil
	}

	// Guard: skip if --simple / bare mode (truthy only — CLAUDE_CODE_SIMPLE=0 is off,
	// same as claudemd.IsAutoMemoryEnabled and querycontext.BareModeFromEnv).
	if querycontext.IsEnvTruthy(os.Getenv("CLAUDE_CODE_SIMPLE")) {
		fileExtractMemoriesLogf("skip reason=simple_mode CLAUDE_CODE_SIMPLE_truthy")
		return nil, nil
	}

	cwd := strings.TrimSpace(p.Cwd)
	if cwd == "" {
		cwd = "."
	}
	memoryDir := claudemd.GetAutoMemPath(cwd)
	if memoryDir == "" {
		fileExtractMemoriesLogf("skip reason=empty_memory_dir cwd=%q", cwd)
		return nil, nil
	}
	fileExtractMemoriesLogf("memory_dir=%q", memoryDir)

	_ = claudemd.EnsureMemoryDirExists(memoryDir)

	throttle := growthbook.GetTenguBrambleLintel()

	// Guard: turn throttle — don't run extraction on every turn.
	state.mu.Lock()
	state.TurnsSinceLastExtraction++
	turnCount := state.TurnsSinceLastExtraction
	state.mu.Unlock()
	if turnCount < throttle {
		fileExtractMemoriesLogf("skip reason=throttle turn_since_extraction=%d throttle=%d", turnCount, throttle)
		return nil, nil
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
		// Advance cursor to avoid re-processing these messages (TS line 348-359).
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

// newMessagesSinceCursor returns messages newer than the cursor UUID.
// If cursor is empty, returns the last 10 messages (first-run heuristic matching TS).
func newMessagesSinceCursor(messages []types.Message, cursorUUID string) []types.Message {
	if cursorUUID == "" {
		// First run: take the last 10 messages.
		if len(messages) <= 10 {
			return messages
		}
		out := make([]types.Message, 10)
		copy(out, messages[len(messages)-10:])
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

	// Derive cwd from memory context.
	memoryDir := ""
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
			if memoryDir == "" {
				// Lazily initialise from cwd.
				cwd, _ := os.Getwd()
				memoryDir = claudemd.GetAutoMemPath(cwd)
			}
			if memoryDir == "" {
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
			if claudemd.IsAutoMemPath(input.FilePath, "") {
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

// runExtractionSubagent runs a lightweight sub-agent via query.Query() with
// restricted tool access.  It builds the extraction prompt from newMessages
// and returns the assistant messages produced by the sub-agent.
func runExtractionSubagent(ctx context.Context, p ExtractionParams, memoryDir string, newMessages []types.Message) ([]string, error) {
	subStart := time.Now()
	fileExtractMemoriesLogf("subagent query start new_messages=%d max_turns=%d", len(newMessages), maxExtractionTurns)
	defer func() {
		fileExtractMemoriesLogf("subagent query end duration_ms=%d", time.Since(subStart).Milliseconds())
	}()

	// Build the extraction prompt.
	prompt := buildExtractionPrompt(p, newMessages, memoryDir)

	// Build sub-agent messages: just the extraction prompt as a user message.
	newUUID := p.NewUUID
	if newUUID == nil {
		newUUID = query.RandomUUID
	}
	userMsg := buildExtractionUserMessage(prompt, newUUID)
	msgs := []types.Message{userMsg}

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
	qdeps := query.ProductionDeps()
	qdeps.NewUUID = newUUID
	qdeps.ToolexecutionDeps = buildRestrictedExecutionDeps(memoryDir)

	qp := query.QueryParams{
		Messages:        msgs,
		SystemPrompt:    p.SystemPrompt,
		ToolUseContext:  tc,
		QuerySource:     types.QuerySource("extract_memories"),
		StreamingParity: true,
		MaxTurns:        &maxTurns,
		Deps:            &qdeps,
	}

	// Collect all yielded messages from the sub-agent query.
	var assistantMessages []types.Message
	for y, err := range query.Query(ctx, qp) {
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

// DrainPendingExtraction waits for any in-flight extraction to complete.
// Mirrors TS drainPendingExtraction in extractMemories.ts.
// In Go, extraction is synchronous (runs inside OnQueryComplete), so this is a
// no-op. It exists for API parity.
func DrainPendingExtraction(timeoutMs ...int) {
	// Extraction is synchronous — nothing to drain.
}

// buildExtractionUserMessage creates a user message with the extraction prompt.
func buildExtractionUserMessage(prompt string, newUUID func() string) types.Message {
	uuid := newUUID()
	content := map[string]any{
		"role":    "user",
		"content": prompt,
	}
	b, _ := json.Marshal(content)
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    uuid,
		Message: b,
	}
}

// buildRestrictedExecutionDeps returns ToolexecutionDeps that restrict tools
// for the extraction sub-agent.
func buildRestrictedExecutionDeps(memoryDir string) toolexecution.ExecutionDeps {
	memDir := strings.TrimSpace(memoryDir)
	invokeReadFileState := localtools.NewReadFileState()

	return toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, _ string, input json.RawMessage) (string, bool, error) {
			switch name {
			case "Read":
				return localtools.ReadFromJSON(input, nil, invokeReadFileState, nil)
			case "Glob":
				return localtools.GlobFromJSON(ctx, input, nil)
			case "Grep":
				return localtools.GrepFromJSON(ctx, input, nil)
			case "Bash":
				// Read-only bash: set GOU_DEMO_READONLY_BASH-like env.
				return localtools.BashFromJSON(ctx, input, "", true)
			case "Write":
				if memDir == "" || !isPathInMemDir(input, memDir) {
					return "", false, fmt.Errorf("Write: path not in memory directory")
				}
				return localtools.WriteFromJSONDeps(input, nil, invokeReadFileState, nil)
			case "Edit":
				if memDir == "" || !isPathInMemDir(input, memDir) {
					return "", false, fmt.Errorf("Edit: path not in memory directory")
				}
				return localtools.EditFromJSONDeps(input, nil, invokeReadFileState, false, nil)
			default:
				return "", false, fmt.Errorf("tool %q not allowed in extraction sub-agent", name)
			}
		},
	}
}

// isPathInMemDir checks whether the file_path in the tool input JSON is inside
// the memory directory.
func isPathInMemDir(input json.RawMessage, memDir string) bool {
	var v struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(input, &v); err != nil || v.FilePath == "" {
		return false
	}
	cleanMem := filepath.Clean(memDir)
	cleanPath := filepath.Clean(v.FilePath)
	rel, err := filepath.Rel(cleanMem, cleanPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
