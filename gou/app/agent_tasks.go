package app

import (
	"sync"
	"time"
)

// AgentTaskState mirrors TS LocalAgentTaskState fields used by CoordinatorTaskPanel.
type AgentTaskState struct {
	ID              string
	AgentType       string
	Description     string
	Name            string // SendMessage routing name
	Status          string // "running", "completed", "killed", "failed"
	StartTime       time.Time
	EndTime         *time.Time
	Progress        *AgentTaskProgress
	EvictAfter      *time.Time // auto-remove after this time (completed/killed)
	IsBackground    bool
	ParentToolUseID string // tool_use_id of the parent Agent tool call
}

// AgentTaskProgress mirrors TS progress fields from LocalAgentTaskState.progress.
type AgentTaskProgress struct {
	TokenCount       int
	ToolUseCount     int
	LastActivity     *time.Time // nil -> arrow up (sending); set -> arrow down (receiving)
	LastActivityDesc string
	Summary          string // from background summarization goroutine
}

// agentTaskStore is the in-memory registry of active/completed agent tasks.
type agentTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*AgentTaskState
}

func newAgentTaskStore() *agentTaskStore {
	return &agentTaskStore{tasks: make(map[string]*AgentTaskState)}
}

func (s *agentTaskStore) Register(task *AgentTaskState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

func (s *agentTaskStore) UpdateProgress(id string, p *AgentTaskProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok && t.Status == "running" {
		t.Progress = p
	}
}

func (s *agentTaskStore) Complete(id string, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		t.Status = status
		now := time.Now()
		t.EndTime = &now
		evict := now.Add(30 * time.Second)
		t.EvictAfter = &evict
	}
}

func (s *agentTaskStore) Kill(id string) {
	s.Complete(id, "killed")
}

func (s *agentTaskStore) Evict(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
}

// EvictExpired removes tasks past their evictAfter deadline. Returns count evicted.
func (s *agentTaskStore) EvictExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, t := range s.tasks {
		if t.EvictAfter != nil && !now.Before(*t.EvictAfter) {
			delete(s.tasks, id)
			n++
		}
	}
	return n
}

// VisibleTasks returns tasks that should be shown in the coordinator panel,
// sorted by start time ascending. Excludes entries with nil/zero EvictAfter.
func (s *agentTaskStore) VisibleTasks() []*AgentTaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AgentTaskState
	for _, t := range s.tasks {
		if t.EvictAfter == nil || t.EvictAfter.IsZero() {
			continue
		}
		out = append(out, t)
	}
	// Sort by start time ascending
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j].StartTime.Before(out[i].StartTime) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Count returns the number of currently tracked tasks.
func (s *agentTaskStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}
