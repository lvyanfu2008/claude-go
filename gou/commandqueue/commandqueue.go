// Command queue for background agent notifications.
// Matches TS messageQueueManager + task-notification XML format.

package commandqueue

import (
	"fmt"
	"strings"
	"sync"
)

// QueuePriority matches TS PRIORITY_ORDER.
type QueuePriority int

const (
	PriorityNow   QueuePriority = 0
	PriorityNext  QueuePriority = 1
	PriorityLater QueuePriority = 2
)

// QueuedCommand is a single entry in the command queue.
type QueuedCommand struct {
	Value    string        // XML notification body
	Mode     string        // "task-notification"
	Priority QueuePriority
}

var (
	commandQueue   []QueuedCommand
	commandQueueMu sync.Mutex
	notifyCh       = make(chan struct{}, 1) // signaled when a command is enqueued
)

// NotifyChan returns a channel that receives when a command is enqueued.
func NotifyChan() <-chan struct{} { return notifyCh }

func signalNotify() {
	select {
	case notifyCh <- struct{}{}:
	default:
	}
}

// EnqueuePendingNotification adds a task-notification command at "later" priority.
func EnqueuePendingNotification(value string) {
	commandQueueMu.Lock()
	defer commandQueueMu.Unlock()
	commandQueue = append(commandQueue, QueuedCommand{
		Value:    value,
		Mode:     "task-notification",
		Priority: PriorityLater,
	})
	signalNotify()
}

// DequeueCommand removes and returns the highest-priority command (FIFO within priority).
func DequeueCommand() *QueuedCommand {
	commandQueueMu.Lock()
	defer commandQueueMu.Unlock()
	if len(commandQueue) == 0 {
		return nil
	}
	// Find highest-priority entry
	bestIdx := 0
	bestPri := commandQueue[0].Priority
	for i := 1; i < len(commandQueue); i++ {
		if commandQueue[i].Priority < bestPri {
			bestPri = commandQueue[i].Priority
			bestIdx = i
		}
	}
	cmd := commandQueue[bestIdx]
	commandQueue = append(commandQueue[:bestIdx], commandQueue[bestIdx+1:]...)
	return &cmd
}

// DrainCommandQueue removes and returns all commands in priority order.
func DrainCommandQueue() []QueuedCommand {
	var out []QueuedCommand
	for {
		cmd := DequeueCommand()
		if cmd == nil {
			break
		}
		out = append(out, *cmd)
	}
	return out
}

// HasPendingNotifications returns true if any task-notification commands are queued.
func HasPendingNotifications() bool {
	commandQueueMu.Lock()
	defer commandQueueMu.Unlock()
	for _, c := range commandQueue {
		if c.Mode == "task-notification" {
			return true
		}
	}
	return false
}

// ClearCommandQueue removes all queued commands.
func ClearCommandQueue() {
	commandQueueMu.Lock()
	defer commandQueueMu.Unlock()
	commandQueue = nil
}

// BuildAgentNotification builds task-notification XML matching TS format.
func BuildAgentNotification(taskID, toolUseID, outputFile, status, summary, result string, tokenCount, toolUseCount int, durationMs int64) string {
	var b strings.Builder
	b.WriteString("<task-notification>\n")
	b.WriteString(fmt.Sprintf("<task-id>%s</task-id>\n", taskID))
	if toolUseID != "" {
		b.WriteString(fmt.Sprintf("<tool-use-id>%s</tool-use-id>\n", toolUseID))
	}
	b.WriteString(fmt.Sprintf("<output-file>%s</output-file>\n", outputFile))
	b.WriteString(fmt.Sprintf("<status>%s</status>\n", status))
	b.WriteString(fmt.Sprintf("<summary>%s</summary>\n", summary))
	if result != "" {
		b.WriteString(fmt.Sprintf("<result>%s</result>\n", result))
	}
	if tokenCount > 0 || toolUseCount > 0 {
		b.WriteString("<usage>\n")
		if tokenCount > 0 {
			b.WriteString(fmt.Sprintf("  <total_tokens>%d</total_tokens>\n", tokenCount))
		}
		if toolUseCount > 0 {
			b.WriteString(fmt.Sprintf("  <tool_uses>%d</tool_uses>\n", toolUseCount))
		}
		if durationMs > 0 {
			b.WriteString(fmt.Sprintf("  <duration_ms>%d</duration_ms>\n", durationMs))
		}
		b.WriteString("</usage>\n")
	}
	b.WriteString("</task-notification>")
	return b.String()
}

// EnqueueAgentNotification is the callback wired into AgentRuntimeConfig.NotificationCallback.
// It builds XML and enqueues at "later" priority.
func EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64) {
	xml := BuildAgentNotification(agentID, toolUseID, outputFile, status, summary, output, tokenCount, toolUseCount, durationMs)
	EnqueuePendingNotification(xml)
}
