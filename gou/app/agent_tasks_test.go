package app

import (
	"testing"
	"time"
)

func TestAgentTaskRegisterAndVisible(t *testing.T) {
	store := newAgentTaskStore()
	task := &AgentTaskState{
		ID:          "agent-1",
		AgentType:   "general-purpose",
		Description: "test task",
		Status:      "running",
		StartTime:   time.Now(),
		EvictAfter:  ptrTime(time.Now().Add(5 * time.Minute)),
	}
	store.Register(task)
	if store.Count() != 1 {
		t.Fatalf("expected 1 task, got %d", store.Count())
	}
	visible := store.VisibleTasks()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible, got %d", len(visible))
	}
}

func TestAgentTaskCompleteAndEvict(t *testing.T) {
	store := newAgentTaskStore()
	task := &AgentTaskState{
		ID:        "agent-1",
		Status:    "running",
		StartTime: time.Now(),
	}
	store.Register(task)

	store.mu.Lock()
	task.EvictAfter = ptrTime(time.Now().Add(-1 * time.Second))
	store.mu.Unlock()

	n := store.EvictExpired(time.Now())
	if n != 1 {
		t.Fatalf("expected 1 evicted, got %d", n)
	}
	if store.Count() != 0 {
		t.Fatalf("expected 0 after evict, got %d", store.Count())
	}
}

func TestAgentTaskProgressUpdate(t *testing.T) {
	store := newAgentTaskStore()
	task := &AgentTaskState{
		ID:     "agent-1",
		Status: "running",
	}
	store.Register(task)

	p := &AgentTaskProgress{TokenCount: 100, ToolUseCount: 3}
	store.UpdateProgress("agent-1", p)

	store.mu.RLock()
	got := store.tasks["agent-1"].Progress
	store.mu.RUnlock()
	if got == nil || got.TokenCount != 100 {
		t.Fatalf("expected TokenCount=100, got %+v", got)
	}
}

func TestAgentTaskKillStatus(t *testing.T) {
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{ID: "a", Status: "running", StartTime: time.Now()})
	store.Kill("a")
	store.mu.RLock()
	task := store.tasks["a"]
	store.mu.RUnlock()
	if task.Status != "killed" {
		t.Fatalf("expected killed, got %s", task.Status)
	}
	if task.EvictAfter == nil {
		t.Fatal("expected EvictAfter set")
	}
}

func TestAgentTaskVisibleExcludesZeroEvictAfter(t *testing.T) {
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{ID: "a", Status: "running", StartTime: time.Now(), EvictAfter: ptrTime(time.Now().Add(5 * time.Minute))})
	store.Register(&AgentTaskState{ID: "b", Status: "running", StartTime: time.Now()})
	visible := store.VisibleTasks()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible, got %d", len(visible))
	}
	if visible[0].ID != "a" {
		t.Fatalf("expected task a visible")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
