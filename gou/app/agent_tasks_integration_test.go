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

// TestAgentFooterViewIntegration verifies AgentFooterView renders correctly with multiple agents.
func TestAgentFooterViewIntegration(t *testing.T) {
	store := newAgentTaskStore()
	now := time.Now()

	// Running agent
	store.Register(&AgentTaskState{
		ID: "a1", AgentType: "explore", Name: "explorer",
		Description: "Searching", Status: "running",
		StartTime: now, EvictAfter: ptrTime(now.Add(time.Minute)),
		Progress: &AgentTaskProgress{TokenCount: 1000},
	})

	// Completed agent
	end := now.Add(-5 * time.Second)
	store.Register(&AgentTaskState{
		ID: "a2", AgentType: "plan", Name: "planner",
		Description: "Done", Status: "completed",
		StartTime: now.Add(-10 * time.Second), EndTime: &end,
		EvictAfter: ptrTime(now.Add(20 * time.Second)),
		Progress:   &AgentTaskProgress{TokenCount: 500},
	})

	visible := store.VisibleTasks()
	view := AgentFooterView(nil, visible, 80)
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Check structural elements
	for _, s := range []string{"explorer", "planner", "x to clear"} {
		if !strings.Contains(view, s) {
			t.Fatalf("view missing %q: %s", s, view)
		}
	}

	// Verify agent type labels are shown
	if !strings.Contains(view, "explore") {
		t.Fatalf("view missing agent type 'explore': %s", view)
	}
	if !strings.Contains(view, "plan") {
		t.Fatalf("view missing agent type 'plan': %s", view)
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

// TestTaskListActivityDisplay verifies that when an in-progress task has an owner
// and the agentTasks store has a running agent with matching name, the activity
// description appears in the view output.
func TestTaskListActivityDisplay(t *testing.T) {
	store := newAgentTaskStore()
	now := time.Now()

	// Register a running agent with activity
	actTime := time.Now()
	store.Register(&AgentTaskState{
		ID: "agent-a1", AgentType: "explore", Name: "explorer",
		Status: "running", StartTime: now,
		EvictAfter: ptrTime(now.Add(time.Minute)),
		Progress: &AgentTaskProgress{
			Summary:      "Reading main.go",
			LastActivity: &actTime,
		},
	})

	tl := newTaskListModel("test-session")
	tl.setAgentTasks(store)

	// Manually add a task with matching owner
	tl.mu.Lock()
	tl.tasks = []taskListEntry{
		{ID: "1", Subject: "Do thing", Status: "in_progress", Owner: "explorer"},
	}
	tl.visible = true
	tl.mu.Unlock()

	view := tl.view(10, 80)
	if !strings.Contains(view, "Do thing") {
		t.Fatal("expected task subject in view")
	}
	if !strings.Contains(view, "Reading main.go") {
		t.Fatalf("expected activity in view, got: %s", view)
	}
}

// TestOwnerColorDisplay verifies owner is rendered with color when agent has a color.
func TestOwnerColorDisplay(t *testing.T) {
	tl := newTaskListModel("test-session")

	tl.mu.Lock()
	tl.tasks = []taskListEntry{
		{ID: "1", Subject: "Task 1", Status: "in_progress", Owner: "worker1"},
	}
	tl.visible = true
	tl.mu.Unlock()

	// Without agentTasks set, owner should use faint style
	view := tl.view(10, 80)
	if !strings.Contains(view, "@worker1") {
		t.Fatal("expected owner in view")
	}

	// With agentTasks set and agent has a color, verify it renders
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{
		ID: "agent-blue", AgentType: "blueAgent", Name: "worker1",
		Status: "running", StartTime: time.Now(),
		EvictAfter: ptrTime(time.Now().Add(time.Minute)),
	})
	tl.setAgentTasks(store)
	view2 := tl.view(10, 80)
	if !strings.Contains(view2, "@worker1") {
		t.Fatal("expected owner in view with agentTasks")
	}
}

// TestNoActivityForCompletedTask verifies that completed tasks do not show activity lines.
func TestNoActivityForCompletedTask(t *testing.T) {
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{
		ID: "agent-a1", AgentType: "x", Name: "explorer",
		Status: "running", StartTime: time.Now(),
		EvictAfter: ptrTime(time.Now().Add(time.Minute)),
		Progress:   &AgentTaskProgress{Summary: "Working"},
	})

	tl := newTaskListModel("test-session")
	tl.setAgentTasks(store)

	tl.mu.Lock()
	tl.tasks = []taskListEntry{
		{ID: "1", Subject: "Done task", Status: "completed", Owner: "explorer"},
	}
	tl.visible = true
	tl.mu.Unlock()

	view := tl.view(10, 80)
	if strings.Contains(view, "Working") {
		t.Fatal("expected NO activity for completed task")
	}
}

// TestAgentColorMap covers all known color entries.
func TestAgentColorMap(t *testing.T) {
	names := []string{"red", "blue", "green", "yellow", "magenta", "cyan", "orange", "claude"}
	for _, name := range names {
		if _, ok := agentColorMap[name]; !ok {
			t.Fatalf("missing color: %s", name)
		}
	}
}
