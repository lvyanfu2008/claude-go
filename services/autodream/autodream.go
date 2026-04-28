// Package autodream mirrors src/services/autoDream/autoDream.ts.
//
// After each query-loop turn (from stopHooks), autoDream runs background memory
// consolidation — the /dream prompt — as a restricted sub-agent when all gates
// pass: time → sessions → lock.
package autodream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/conversation-runtime/query"
	"goc/memdir"
	"goc/tools/localtools"
	"goc/tools/toolexecution"
	"goc/types"
)

// Scan interval when time-gate passes but session-gate doesn't.
const sessionScanIntervalMs = 10 * 60 * 1000 // 10 minutes

// Max turns for the consolidation sub-agent.
const maxConsolidationTurns = 5

// State holds autoDream state across query turns.
type State struct {
	mu              sync.Mutex
	lastSessionScan int64 // epoch ms
}

// NewState creates a fresh autoDream State.
func NewState() *State {
	return &State{}
}

// Execute runs one auto-dream pass.
//
// Mirrors executeAutoDream / runAutoDream in autoDream.ts.
//
// Parameters:
//   - ctx: context for cancellation
//   - state: persistent state across turns (scan throttle)
//   - toolUseContext: inherited from parent (for sub-agent config)
//   - systemPrompt: inherited from parent
//   - userContext: key-value pairs
//   - systemContext: key-value pairs
//   - querySource: query source label
//   - newUUID: UUID generator (nil = use RandomUUID)
//   - configHome: ~/.claude or CLAUDE_CONFIG_DIR
//   - originalCwd: the original working directory
//   - memoryDir: the auto-memory directory (empty = resolve from originalCwd)
//   - currentSessionID: current session ID to exclude from session count
//
// Returns written memory file paths (excluding MEMORY.md), or nil if skipped.
func Execute(
	ctx context.Context,
	state *State,
	toolUseContext types.ToolUseContext,
	systemPrompt query.SystemPrompt,
	userContext map[string]string,
	systemContext map[string]string,
	querySource types.QuerySource,
	newUUID func() string,
	configHome string,
	originalCwd string,
	memoryDir string,
	currentSessionID string,
) ([]string, error) {
	// Guard: auto-dream must be enabled.
	if !IsAutoDreamEnabled() {
		return nil, nil
	}

	// Guard: auto-memory must be enabled.
	if !memdir.IsAutoMemoryEnabled() {
		return nil, nil
	}

	if memoryDir == "" {
		memoryDir = memdir.GetAutoMemPath(originalCwd)
		if memoryDir == "" {
			return nil, nil
		}
	}

	// Ensure memory dir exists.
	_ = os.MkdirAll(memoryDir, 0755)

	cfg := GetConfig()
	if cfg.MinHours <= 0 {
		cfg.MinHours = DefaultAutoDreamConfig.MinHours
	}
	if cfg.MinSessions <= 0 {
		cfg.MinSessions = DefaultAutoDreamConfig.MinSessions
	}

	// --- Time gate ---
	lastAt, err := ReadLastConsolidatedAt(memoryDir)
	if err != nil {
		log.Printf("[autoDream] ReadLastConsolidatedAt failed: %v", err)
		return nil, nil
	}
	hoursSince := float64(time.Now().UnixMilli()-lastAt) / 3_600_000
	if hoursSince < float64(cfg.MinHours) {
		return nil, nil
	}

	// --- Scan throttle ---
	state.mu.Lock()
	now := time.Now().UnixMilli()
	sinceScanMs := now - state.lastSessionScan
	if sinceScanMs < sessionScanIntervalMs {
		state.mu.Unlock()
		log.Printf("[autoDream] scan throttle — time-gate passed but last scan was %ds ago", sinceScanMs/1000)
		return nil, nil
	}
	state.lastSessionScan = now
	state.mu.Unlock()

	// --- Session gate ---
	sessionIDs, err := ListSessionsTouchedSince(lastAt, originalCwd, configHome)
	if err != nil {
		log.Printf("[autoDream] ListSessionsTouchedSince failed: %v", err)
		return nil, nil
	}
	// Exclude current session.
	if currentSessionID != "" {
		filtered := sessionIDs[:0]
		for _, id := range sessionIDs {
			if id != currentSessionID {
				filtered = append(filtered, id)
			}
		}
		sessionIDs = filtered
	}
	if len(sessionIDs) < cfg.MinSessions {
		log.Printf("[autoDream] skip — %d sessions since last consolidation, need %d", len(sessionIDs), cfg.MinSessions)
		return nil, nil
	}

	// --- Lock ---
	priorMtime, err := TryAcquireConsolidationLock(memoryDir)
	if err != nil {
		log.Printf("[autoDream] lock acquire failed: %v", err)
		return nil, nil
	}
	if priorMtime < 0 {
		return nil, nil
	}

	log.Printf("[autoDream] firing — %.1fh since last, %d sessions to review", hoursSince, len(sessionIDs))

	// Run the consolidation sub-agent.
	writtenPaths, err := runConsolidationSubagent(ctx, memoryDir, systemPrompt, toolUseContext, userContext, systemContext, querySource, newUUID, originalCwd, configHome, sessionIDs)
	if err != nil {
		log.Printf("[autoDream] consolidation failed: %v", err)
		RollbackConsolidationLock(memoryDir, priorMtime)
		return nil, nil
	}

	// Filter out MEMORY.md.
	memoryPaths := filterMemoryPaths(writtenPaths, memoryDir)

	log.Printf("[autoDream] completed — %d memory files written", len(memoryPaths))

	return memoryPaths, nil
}

// runConsolidationSubagent runs the consolidation prompt as a restricted sub-agent.
func runConsolidationSubagent(
	ctx context.Context,
	memoryDir string,
	systemPrompt query.SystemPrompt,
	toolUseContext types.ToolUseContext,
	userContext map[string]string,
	systemContext map[string]string,
	querySource types.QuerySource,
	newUUID func() string,
	originalCwd string,
	configHome string,
	sessionIDs []string,
) ([]string, error) {
	if newUUID == nil {
		newUUID = query.RandomUUID
	}

	// Build the consolidation prompt.
	transcriptDir := ProjectDirForOriginalCwd(originalCwd, configHome)
	extra := buildExtraSection(sessionIDs)
	prompt := BuildConsolidationPrompt(memoryDir, transcriptDir, extra)

	userMsg := buildUserMessage(prompt, newUUID)
	msgs := []types.Message{userMsg}

	// Clone tool context for sub-agent.
	tc := toolUseContext
	tc.AgentID = nil
	tc.Options.IsNonInteractiveSession = true

	maxTurns := maxConsolidationTurns

	qdeps := query.ProductionDeps()
	qdeps.NewUUID = newUUID
	qdeps.ToolexecutionDeps = buildRestrictedExecutionDeps(memoryDir)

	qp := query.QueryParams{
		Messages:        msgs,
		SystemPrompt:    systemPrompt,
		ToolUseContext:  tc,
		QuerySource:     types.QuerySource("auto_dream"),
		StreamingParity: true,
		MaxTurns:        &maxTurns,
		Deps:            &qdeps,
	}

	var assistantMessages []types.Message
	for y, err := range query.Query(ctx, qp) {
		if err != nil {
			return nil, err
		}
		if y.Message != nil && y.Message.Type == types.MessageTypeAssistant {
			assistantMessages = append(assistantMessages, *y.Message)
		}
	}

	return extractWrittenPaths(assistantMessages), nil
}

// buildExtraSection creates the extra context section for the consolidation prompt,
// including the tool constraints notice and session list.
func buildExtraSection(sessionIDs []string) string {
	var b strings.Builder
	b.WriteString("\n\n**Tool constraints for this run:** Bash is restricted to read-only commands (`ls`, `find`, `grep`, `cat`, `stat`, `wc`, `head`, `tail`, and similar). Anything that writes, redirects to a file, or modifies state will be denied. Plan your exploration with this in mind — no need to probe.\n\n")
	b.WriteString(fmt.Sprintf("Sessions since last consolidation (%d):\n", len(sessionIDs)))
	for _, id := range sessionIDs {
		b.WriteString(fmt.Sprintf("- %s\n", id))
	}
	return strings.TrimSpace(b.String())
}

// buildUserMessage creates a user message with the given prompt content.
func buildUserMessage(prompt string, newUUID func() string) types.Message {
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

// buildRestrictedExecutionDeps returns toolexecution.ExecutionDeps that restrict
// tools for the consolidation sub-agent (same as extractmemories).
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
				return "", false, fmt.Errorf("tool %q not allowed in consolidation sub-agent", name)
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

// extractWrittenPaths scans assistant messages for Write/Edit tool_use blocks
// and returns the file_path values.
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
