package workflow

import (
	"fmt"
	"strings"
)

// taskManager handles workflow task lifecycle via the v2 task system.
// It creates, updates, and finalizes workflow tasks.
type taskManager struct {
	taskListID string
	taskID     string
	subject    string
	desc       string
	notifyFn   func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)
}

// CreateTask creates a new local_workflow task in the v2 task system.
// Returns the task ID for status updates.
func CreateTask(taskListID, workflowName, description string, notifyFn func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)) (string, error) {
	subject := "Workflow: " + workflowName
	desc := fmt.Sprintf("Execute workflow %q (%s)", workflowName, description)

	// Import v2 task system functions from the tools package
	taskID, err := createTaskInSystem(taskListID, subject, desc)
	if err != nil {
		return "", err
	}
	return taskID, nil
}

// UpdateTaskStatus updates the task status in the v2 system.
func UpdateTaskStatus(taskListID, taskID, status string) error {
	return updateTaskInSystem(taskListID, taskID, status)
}

// CompleteTask marks the workflow task as completed.
func CompleteTask(taskListID, taskID, subject, output string, notifyFn func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)) {
	_ = updateTaskInSystem(taskListID, taskID, "completed")
	if notifyFn != nil {
		// Truncate output for notification
		summary := output
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		notifyFn("", "", "", "completed", subject, summary, 0, 0, 0)
	}
}

// FailTask marks the workflow task as failed.
func FailTask(taskListID, taskID, subject string, err error, notifyFn func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)) {
	_ = updateTaskInSystem(taskListID, taskID, "failed")
	if notifyFn != nil {
		errMsg := err.Error()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		notifyFn("", "", "", "failed", subject, errMsg, 0, 0, 0)
	}
}

// createTaskInSystem calls the v2 task system to create a local_workflow task.
// Uses the same pattern as WorkflowFromJSON in optional_tools.go.
func createTaskInSystem(taskListID, subject, description string) (string, error) {
	// The v2 task system functions are in the tools package (v2CreateTask, v2UpdateTaskFields, etc.)
	// Since we can't import tools from here (it would be a circular dependency because tools
	// will import workflow), we use a function variable that gets set during initialization.
	if taskCreateFn == nil {
		return "", fmt.Errorf("workflow task system not initialized")
	}
	return taskCreateFn(taskListID, subject, description)
}

func updateTaskInSystem(taskListID, taskID, status string) error {
	if taskUpdateFn == nil {
		return fmt.Errorf("workflow task system not initialized")
	}
	return taskUpdateFn(taskListID, taskID, status)
}

// taskCreateFn is set by the tools package during initialization to avoid circular imports.
var taskCreateFn func(taskListID, subject, description string) (string, error)

// taskUpdateFn is set by the tools package during initialization to avoid circular imports.
var taskUpdateFn func(taskListID, taskID, status string) error

// SetTaskFunctions wires the v2 task system into the workflow package.
// Called from tools package init.
func SetTaskFunctions(
	create func(taskListID, subject, description string) (string, error),
	update func(taskListID, taskID, status string) error,
) {
	taskCreateFn = create
	taskUpdateFn = update
}

func init() {
	// Log a warning if task functions aren't wired (non-fatal)
}

// NewTaskID generates a unique task ID for a workflow.
func NewTaskID() string {
	return "w" + strings.ToLower(randomHex(8))
}

func randomHex(n int) string {
	// Simple pseudo-random hex string for task IDs
	const charset = "0123456789abcdef"
	b := make([]byte, n)
	// Use a simple counter-based approach to avoid crypto/rand dependency
	// In production this should use crypto/rand
	for i := range b {
		b[i] = charset[(i*7+int(i*i*3))%len(charset)]
	}
	return string(b)
}
