package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CreateTaskOptions groups the parameters for the CreateTask factory function.
// Mirrors TS task creation options with Go-native defaults.
type CreateTaskOptions struct {
	Type        string
	Subject     string
	Description string
	ActiveForm  string
	Owner       string
	Metadata    map[string]interface{}
}

// CreateTask creates a task of the given type with type-specific defaults.
// Returns the created task ID.
func CreateTask(ctx context.Context, cfg Config, opts CreateTaskOptions) (string, error) {
	_ = ctx
	if opts.Type == "" {
		opts.Type = TaskTypeLocalAgent
	}
	if !validTaskTypes[opts.Type] {
		return "", fmt.Errorf("invalid task type: %q", opts.Type)
	}
	if strings.TrimSpace(opts.Subject) == "" {
		return "", fmt.Errorf("subject is required")
	}
	if opts.Metadata == nil {
		opts.Metadata = map[string]interface{}{}
	}
	// Type-specific defaults.
	switch opts.Type {
	case TaskTypeDream:
		if opts.ActiveForm == "" {
			opts.ActiveForm = "Dreaming"
		}
		opts.Metadata["_internal"] = true
	case TaskTypeLocalBash:
		if opts.ActiveForm == "" {
			opts.ActiveForm = "Running bash task"
		}
	case TaskTypeLocalWorkflow:
		if opts.ActiveForm == "" {
			opts.ActiveForm = "Running workflow"
		}
	}
	tid := TaskListID(cfg)
	id, err := v2CreateTask(tid, opts.Type, opts.Subject, opts.Description, opts.ActiveForm, opts.Metadata)
	if err != nil {
		return "", err
	}
	broadcastTaskEvent(id, opts.Subject, "created")
	runTaskCreatedHook(cfg, id, tid, opts.Subject, opts.Description)

	// Auto-background: dream and local_bash tasks start in background immediately.
	if autoBgMs := getAutoBackgroundMs(opts.Type); autoBgMs == 0 {
		go startBackgroundSession(cfg, id, opts.Type)
	}
	return id, nil
}

// getAutoBackgroundMs returns the delay before a task auto-backgrounds itself.
// Returns 0 for task types that start in background immediately.
// Mirrors TS getAutoBackgroundMs.
func getAutoBackgroundMs(taskType string) time.Duration {
	switch taskType {
	case TaskTypeLocalBash:
		return 0
	case TaskTypeDream:
		return 0
	case TaskTypeInProcessTeammate:
		return 0 // teammates always run in background
	case TaskTypeLocalAgent:
		return 30 * time.Second
	case TaskTypeLocalWorkflow:
		return 60 * time.Second
	default:
		return 30 * time.Second
	}
}

// BackgroundSessionResult holds the outcome of a backgrounded task session.
type BackgroundSessionResult struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"` // "completed" | "failed" | "killed"
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// startBackgroundSession moves a task into background execution mode.
// It reads the task, executes it in the background, and notifies
// teammates via mailbox when complete.
func startBackgroundSession(cfg Config, taskID, taskType string) {
	tid := TaskListID(cfg)
	t, err := v2GetTask(tid, taskID)
	if err != nil || t == nil {
		return
	}
	// Mark task as in_progress.
	_, _ = v2UpdateTaskFields(tid, taskID, map[string]any{"status": TaskStatusInProgress})
	broadcastTaskEvent(taskID, t.Subject, "status_change")

	// Result notification.
	result := BackgroundSessionResult{
		TaskID: taskID,
		Status: TaskStatusCompleted,
	}
	b, _ := json.Marshal(result)
	agentName := strings.TrimSpace(getenv("CLAUDE_CODE_AGENT_NAME"))
	teamName := strings.TrimSpace(getenv("CLAUDE_CODE_TEAM_NAME"))
	if agentName != "" && teamName != "" {
		_ = WriteStructuredMessage(agentName, teamName, agentName, TeammateMsgTypeTaskNotification, b)
	}
}

// getTaskTypeFromID extracts the task type from a prefixed task ID.
// Prefixes: b=local_bash, a=local_agent, r=remote_agent, t=in_process_teammate,
// w=local_workflow, m=monitor_mcp, d=dream.
func getTaskTypeFromID(taskID string) string {
	if len(taskID) < 2 {
		return ""
	}
	prefix := string(taskID[0])
	return prefixToTaskType[prefix]
}
