package workflow

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"goc/types"
)

// Meta is extracted from `export const meta = {...}` in the workflow script.
// Must be a pure literal — no variables, function calls, spreads, or template interpolation.
type Meta struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Phases      []PhaseMeta `json:"phases,omitempty"`
}

// PhaseMeta describes a single phase entry in meta.phases.
type PhaseMeta struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Model  string `json:"model,omitempty"`
}

// AgentOpts mirrors the opts parameter for agent() calls in workflow scripts.
type AgentOpts struct {
	Label     string          `json:"label,omitempty"`
	Phase     string          `json:"phase,omitempty"`
	Schema    json.RawMessage `json:"schema,omitempty"`
	Model     string          `json:"model,omitempty"`
	Isolation string          `json:"isolation,omitempty"`
	AgentType string          `json:"agentType,omitempty"`
}

// WorkflowInput mirrors the JSON tool call input for the workflow tool.
type WorkflowInput struct {
	Script          string          `json:"script,omitempty"`
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	Title           string          `json:"title,omitempty"`
	Args            json.RawMessage `json:"args,omitempty"`
	ScriptPath      string          `json:"scriptPath,omitempty"`
	ResumeFromRunID string          `json:"resumeFromRunId,omitempty"`
}

// EngineConfig wires the Go dependencies the workflow engine needs.
type EngineConfig struct {
	WorkDir             string
	ProjectRoot         string
	SessionID           string
	TasksDir            string
	MainLoopModel       string
	AvailableMCPServers []string
	Messages            []types.Message
	SystemPrompt        []string
	Args                json.RawMessage // Passed as the `args` global in workflow scripts
	// ProgressCallback is called for raw progress messages (phase, log).
	ProgressCallback func(*types.Message)
	// WorkflowProgressCallback is called to report workflow node/agent progress to the UI.
	// (agentID, status, message) where agentID is the workflow's run ID.
	WorkflowProgressCallback func(agentID, status, message string)
	// NotificationCallback is called when the workflow completes/fails.
	NotificationCallback func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)
	ToolPermission      *types.ToolPermissionContextData
	ToolUseID           string
}

// RunState tracks the mutable state of a single workflow execution.
type RunState struct {
	RunID        string
	Meta         Meta
	TaskID       string
	Args         json.RawMessage
	CurrentPhase string
	Budget       *BudgetTracker
	Journal      *Journal

	// AgentCount tracks spawned agent count (atomic).
	AgentCount atomic.Int32

	// ProgressCallback forwards progress to the UI layer.
	ProgressCallback func(*types.Message)
	// WorkflowProgressCallback emits per-node progress to the UI.
	WorkflowProgressCallback func(agentID, status, message string)

	// agentSem limits concurrent agent executions (capped at 16).
	agentSem chan struct{}

	// abortCh is closed when the workflow is aborted.
	abortCh chan struct{}

	// mu protects non-atomic state.
	mu sync.Mutex
}

// NewRunState creates a RunState with default concurrency limits.
func NewRunState(runID string, meta Meta, args json.RawMessage, progressCallback func(*types.Message), workflowProgressFn func(agentID, status, message string)) *RunState {
	return &RunState{
		RunID:                    runID,
		Meta:                     meta,
		Args:                     args,
		ProgressCallback:         progressCallback,
		WorkflowProgressCallback: workflowProgressFn,
		Budget:                   NewBudgetTracker(0),
		Journal:                  NewJournal(runID),
		agentSem:                 make(chan struct{}, 16),
		abortCh:                  make(chan struct{}),
	}
}

// emitProgress sends a progress update to the UI via the workflow progress callback.
func (s *RunState) emitProgress(status, message string) {
	if s.WorkflowProgressCallback != nil {
		s.WorkflowProgressCallback(s.RunID, status, message)
	}
}

// AcquireSlot blocks until an agent concurrency slot is available.
// Returns false if the workflow was aborted.
func (s *RunState) AcquireSlot() bool {
	// Check abort first to avoid race with closed channel
	select {
	case <-s.abortCh:
		return false
	default:
	}
	select {
	case s.agentSem <- struct{}{}:
		return true
	case <-s.abortCh:
		return false
	}
}

// ReleaseSlot releases an agent concurrency slot.
func (s *RunState) ReleaseSlot() {
	<-s.agentSem
}

// Aborted returns a channel that closes when the workflow is aborted.
func (s *RunState) Aborted() <-chan struct{} {
	return s.abortCh
}

// Abort signals the workflow to stop.
func (s *RunState) Abort() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.abortCh:
		// Already aborted
	default:
		close(s.abortCh)
	}
}
