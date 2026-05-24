package app

import (
	"strings"
	"testing"
	"time"
)

// TestAgentFullLifecycle verifies register -> progress -> complete -> visible -> evict.
func TestAgentFullLifecycle(t *testing.T) {
	store := newAgentTaskStore()

	// 1. Register
	store.Register(&AgentTaskState{
		ID: "agent-1", AgentType: "test", Description: "desc",
		Status: "running", StartTime: time.Now(),
		EvictAfter: ptrTime(time.Now().Add(time.Minute)),
	})

	// 2. Progress update
	p := &AgentTaskProgress{TokenCount: 500, ToolUseCount: 2, Summary: "Reading main.go"}
	store.UpdateProgress("agent-1", p)

	// Verify progress persisted
	store.mu.RLock()
	got := store.tasks["agent-1"].Progress
	store.mu.RUnlock()
	if got.Summary != "Reading main.go" {
		t.Fatalf("summary mismatch: %q", got.Summary)
	}
	if got.TokenCount != 500 {
		t.Fatalf("TokenCount mismatch: %d", got.TokenCount)
	}

	// 3. Complete
	store.Complete("agent-1", "completed")

	store.mu.RLock()
	task := store.tasks["agent-1"]
	store.mu.RUnlock()
	if task.Status != "completed" {
		t.Fatalf("expected completed, got %s", task.Status)
	}
	if task.EvictAfter == nil {
		t.Fatal("expected evict time set")
	}

	// 4. Still visible (recently completed)
	visible := store.VisibleTasks()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible, got %d", len(visible))
	}

	// 5. Evict after timeout
	store.mu.Lock()
	task.EvictAfter = ptrTime(time.Now().Add(-1 * time.Second))
	store.mu.Unlock()
	n := store.EvictExpired(time.Now())
	if n != 1 {
		t.Fatalf("expected 1 evicted, got %d", n)
	}
	if len(store.VisibleTasks()) != 0 {
		t.Fatal("expected 0 visible after evict")
	}
}

// TestCoordinatorViewIntegration verifies multiple agents in different states render correctly.
func TestCoordinatorViewIntegration(t *testing.T) {
	m := &model{agentTasks: newAgentTaskStore()}
	now := time.Now()

	// Running agent
	m.agentTasks.Register(&AgentTaskState{
		ID: "a1", AgentType: "explore", Name: "explorer",
		Description: "Searching", Status: "running",
		StartTime: now, EvictAfter: ptrTime(now.Add(time.Minute)),
		Progress: &AgentTaskProgress{TokenCount: 1000},
	})

	// Completed agent
	end := now.Add(-5 * time.Second)
	m.agentTasks.Register(&AgentTaskState{
		ID: "a2", AgentType: "plan", Name: "planner",
		Description: "Done", Status: "completed",
		StartTime: now.Add(-10 * time.Second), EndTime: &end,
		EvictAfter: ptrTime(now.Add(20 * time.Second)),
		Progress: &AgentTaskProgress{TokenCount: 500},
	})

	view := m.agentCoordinatorView()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Check major structural elements
	for _, s := range []string{"main", "explorer", "planner", "x to clear"} {
		if !strings.Contains(view, s) {
			t.Fatalf("view missing %q: %s", s, view)
		}
	}

	// Verify lines: main + 2 agents = 3 \n separated lines
	lines := strings.Split(strings.TrimSpace(view), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (main + 2 agents), got %d: %q", len(lines), view)
	}
}

// TestAgentProgressUpdateOnlyRunning ensures progress only updates running agents.
func TestAgentProgressUpdateOnlyRunning(t *testing.T) {
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{ID: "a", Status: "running", StartTime: time.Now()})
	store.Complete("a", "completed")

	// Try updating progress on completed agent -- should be ignored
	p := &AgentTaskProgress{TokenCount: 999}
	store.UpdateProgress("a", p)

	store.mu.RLock()
	got := store.tasks["a"].Progress
	store.mu.RUnlock()
	if got != nil {
		t.Fatal("expected progress to be nil (not updated on completed agent)")
	}
}

// TestAgentEvictExpiredMultipleTasks tests batch eviction.
func TestAgentEvictExpiredMultipleTasks(t *testing.T) {
	store := newAgentTaskStore()
	now := time.Now()

	// Register 3 tasks with different eviction times
	store.Register(&AgentTaskState{
		ID: "a1", Status: "running", StartTime: now,
		EvictAfter: ptrTime(now.Add(time.Hour)), // far future
	})
	store.Register(&AgentTaskState{
		ID: "a2", Status: "completed", StartTime: now,
		EvictAfter: ptrTime(now.Add(-1 * time.Second)), // already expired
	})
	store.Register(&AgentTaskState{
		ID: "a3", Status: "completed", StartTime: now,
		EvictAfter: ptrTime(now.Add(-5 * time.Second)), // already expired
	})

	n := store.EvictExpired(now)
	if n != 2 {
		t.Fatalf("expected 2 evicted, got %d", n)
	}
	if store.Count() != 1 {
		t.Fatalf("expected 1 remaining, got %d", store.Count())
	}
	store.mu.RLock()
	_, ok := store.tasks["a1"]
	store.mu.RUnlock()
	if !ok {
		t.Fatal("expected a1 to survive eviction")
	}
}
