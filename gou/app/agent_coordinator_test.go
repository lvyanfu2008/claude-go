package app

import (
	"strings"
	"testing"
	"time"
	state "goc/gou/app/state"
)

func TestCoordinatorViewEmpty(t *testing.T) {
	store := newAgentTaskStore()
	m := &model{Agent: &state.Agent{Tasks: store}}
	view := m.agentCoordinatorView()
	if view != "" {
		t.Fatalf("expected empty view, got %q", view)
	}
}

func TestCoordinatorViewNilStore(t *testing.T) {
	m := &model{}
	view := m.agentCoordinatorView()
	if view != "" {
		t.Fatalf("expected empty view for nil store, got %q", view)
	}
}

func TestCoordinatorViewWithRunningTask(t *testing.T) {
	store := newAgentTaskStore()
	store.Register(&AgentTaskState{
		ID:          "agent-1",
		AgentType:   "general-purpose",
		Name:        "researcher",
		Description: "Reading runAgent.go",
		Status:      "running",
		StartTime:   time.Now(),
		EvictAfter:  ptrTime(time.Now().Add(5 * time.Minute)),
		Progress:    &AgentTaskProgress{TokenCount: 4200, LastActivity: ptrTime(time.Now())},
	})

	m := &model{Agent: &state.Agent{Tasks: store}}
	view := m.agentCoordinatorView()
	if !strings.Contains(view, "main") {
		t.Fatal("expected 'main' row")
	}
	if !strings.Contains(view, "researcher") {
		t.Fatal("expected 'researcher' in view")
	}
	if !strings.Contains(view, agentPlayIcon) {
		t.Fatal("expected play icon for running task")
	}
	if !strings.Contains(view, "4.2k") {
		t.Fatal("expected 4.2k tokens")
	}
}

func TestCoordinatorViewCompletedTask(t *testing.T) {
	store := newAgentTaskStore()
	endTime := time.Now()
	store.Register(&AgentTaskState{
		ID:          "agent-2",
		AgentType:   "plan",
		Name:        "planner",
		Description: "Finished",
		Status:      "completed",
		StartTime:   time.Now().Add(-10 * time.Second),
		EndTime:     &endTime,
		EvictAfter:  ptrTime(time.Now().Add(20 * time.Second)),
		Progress:    &AgentTaskProgress{TokenCount: 3100},
	})

	m := &model{Agent: &state.Agent{Tasks: store}}
	view := m.agentCoordinatorView()
	if !strings.Contains(view, agentPauseIcon) {
		t.Fatal("expected pause icon for completed task")
	}
	if !strings.Contains(view, "x to clear") {
		t.Fatal("expected 'x to clear' hint")
	}
	if !strings.Contains(view, "3.1k") {
		t.Fatal("expected 3.1k tokens")
	}
}

func TestCoordinatorViewMultipleTasks(t *testing.T) {
	store := newAgentTaskStore()
	now := time.Now()
	store.Register(&AgentTaskState{
		ID: "a1", AgentType: "g1", Name: "n1", Status: "running",
		StartTime: now, EvictAfter: ptrTime(now.Add(time.Minute)),
	})
	store.Register(&AgentTaskState{
		ID: "a2", AgentType: "g2", Name: "n2", Status: "completed",
		StartTime: now.Add(time.Second), EndTime: ptrTime(now.Add(5 * time.Second)),
		EvictAfter: ptrTime(now.Add(time.Minute)),
	})

	m := &model{Agent: &state.Agent{Tasks: store}}
	view := m.agentCoordinatorView()
	lines := strings.Split(strings.TrimSpace(view), "\n")
	// main + 2 task rows = 3 lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), view)
	}
}

func TestFormatAgentElapsed(t *testing.T) {
	tests := []struct {
		name string
		task *AgentTaskState
		want string
	}{
		{
			name: "running zero elapsed",
			task: &AgentTaskState{Status: "running", StartTime: time.Now()},
			want: "0.0s",
		},
		{
			name: "running recent",
			task: &AgentTaskState{Status: "running", StartTime: time.Now().Add(-5 * time.Second)},
			want: "5.0s",
		},
		{
			name: "completed",
			task: &AgentTaskState{Status: "completed", StartTime: time.Now().Add(-10 * time.Second), EndTime: ptrTime(time.Now())},
			want: "10.0s",
		},
		{
			name: "over a minute",
			task: &AgentTaskState{Status: "running", StartTime: time.Now().Add(-2 * time.Minute)},
			want: "2m 0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAgentElapsed(tt.task)
			if got != tt.want {
				t.Errorf("formatAgentElapsed() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatAgentTokens(t *testing.T) {
	tests := []struct {
		name string
		p    *AgentTaskProgress
		want string
	}{
		{
			name: "nil progress",
			p:    nil,
			want: "",
		},
		{
			name: "zero tokens",
			p:    &AgentTaskProgress{TokenCount: 0},
			want: "",
		},
		{
			name: "small count",
			p:    &AgentTaskProgress{TokenCount: 500},
			want: "500 tokens",
		},
		{
			name: "exactly 1000",
			p:    &AgentTaskProgress{TokenCount: 1000},
			want: "1.0k tokens",
		},
		{
			name: "large count",
			p:    &AgentTaskProgress{TokenCount: 4200},
			want: "4.2k tokens",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAgentTokens(tt.p)
			if got != tt.want {
				t.Errorf("formatAgentTokens() = %q, want %q", got, tt.want)
			}
		})
	}
}
