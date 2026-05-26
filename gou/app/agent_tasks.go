package app

import (
	"sync"
	"time"
)

const (
	agentEvictMin    = 2 * time.Second
	agentEvictNormal = 5 * time.Second
	agentEvictError  = 10 * time.Second
	mainSessionID    = "main-session"
)

// AgentTaskState mirrors TS LocalAgentTaskState fields used by CoordinatorTaskPanel.
type AgentTaskState struct {
	ID              string
	AgentType       string
	Description     string
	Name            string
	Status          string // "running", "completed", "killed", "failed", "stopped"
	StartTime       time.Time
	EndTime         *time.Time
	Duration        time.Duration // cached elapsed on completion
	Progress        *AgentTaskProgress
	EvictAfter      *time.Time
	IsBackground    bool
	ParentToolUseID string
	ErrorMessage    string
}

// AgentTaskProgress mirrors TS progress fields.
type AgentTaskProgress struct {
	TokenCount       int
	ToolUseCount     int
	LastActivity     *time.Time
	LastActivityDesc string
	Summary          string
}

// agentTaskStore is the in-memory registry of active/completed agent tasks.
type agentTaskStore struct {
	mu       sync.RWMutex
	tasks    map[string]*AgentTaskState
	mainTask *AgentTaskState
}

func newAgentTaskStore() *agentTaskStore {
	return &agentTaskStore{tasks: make(map[string]*AgentTaskState)}
}

func (s *agentTaskStore) Register(task *AgentTaskState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

// RegisterMainSession creates the main session task entry.
func (s *agentTaskStore) RegisterMainSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mainTask = &AgentTaskState{
		ID:        mainSessionID,
		AgentType: "main-session",
		Status:    "running",
		StartTime: time.Now(),
	}
}

// CompleteMainSession marks the main session as completed.
func (s *agentTaskStore) CompleteMainSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mainTask != nil {
		s.mainTask.Status = "completed"
		now := time.Now()
		s.mainTask.EndTime = &now
		s.mainTask.Duration = now.Sub(s.mainTask.StartTime)
		evict := now.Add(agentEvictNormal)
		s.mainTask.EvictAfter = &evict
	}
}

// StopMainSession marks the main session as stopped.
func (s *agentTaskStore) StopMainSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mainTask != nil && s.mainTask.Status == "running" {
		s.mainTask.Status = "stopped"
		now := time.Now()
		s.mainTask.EndTime = &now
		s.mainTask.Duration = now.Sub(s.mainTask.StartTime)
	}
}

// MainTask returns the main session task (nil if not registered).
func (s *agentTaskStore) MainTask() *AgentTaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mainTask
}

func (s *agentTaskStore) UpdateProgress(id string, p *AgentTaskProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == mainSessionID {
		if s.mainTask != nil && s.mainTask.Status == "running" {
			s.mainTask.Progress = p
		}
		return
	}
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
		t.Duration = now.Sub(t.StartTime)
		deadline := agentEvictNormal
		if status == "failed" {
			deadline = agentEvictError
		}
		if t.Duration < time.Second {
			deadline = agentEvictMin
		}
		evict := now.Add(deadline)
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
	if s.mainTask != nil && s.mainTask.EvictAfter != nil && !now.Before(*s.mainTask.EvictAfter) {
		s.mainTask = nil
		n++
	}
	return n
}

// VisibleTasks returns tasks that should be shown, sorted by start time ascending.
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
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j].StartTime.Before(out[i].StartTime) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// RunningAgents returns non-main-session tasks with status "running".
func (s *agentTaskStore) RunningAgents() []*AgentTaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AgentTaskState
	for _, t := range s.tasks {
		if t.Status == "running" {
			out = append(out, t)
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
