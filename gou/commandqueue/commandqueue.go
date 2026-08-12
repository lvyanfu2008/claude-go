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

// Queue 是 per-session 的命令队列(WS 每连接一个,stdio 用默认单例)。
type Queue struct {
	mu              sync.Mutex
	commands        []QueuedCommand
	notifyCh        chan struct{} // signaled when a command is enqueued
	pendingBgAgents int
}

// NewQueue 创建独立队列实例。
func NewQueue() *Queue {
	return &Queue{notifyCh: make(chan struct{}, 1)}
}

// defaultQueue 是进程级默认队列,stdio 模式复用(向后兼容)。
var defaultQueue = NewQueue()

// NotifyChan returns a channel that receives when a command is enqueued.
func NotifyChan() <-chan struct{} { return defaultQueue.notifyCh }

func (q *Queue) signalNotify() {
	select {
	case q.notifyCh <- struct{}{}:
	default:
	}
}

// EnqueuePendingNotification adds a task-notification command at "later" priority.
func (q *Queue) EnqueuePendingNotification(value string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.commands = append(q.commands, QueuedCommand{
		Value:    value,
		Mode:     "task-notification",
		Priority: PriorityLater,
	})
	q.signalNotify()
}

func EnqueuePendingNotification(value string) { defaultQueue.EnqueuePendingNotification(value) }

// DequeueCommand removes and returns the highest-priority command (FIFO within priority).
func (q *Queue) DequeueCommand() *QueuedCommand {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.commands) == 0 {
		return nil
	}
	// Find highest-priority entry
	bestIdx := 0
	bestPri := q.commands[0].Priority
	for i := 1; i < len(q.commands); i++ {
		if q.commands[i].Priority < bestPri {
			bestPri = q.commands[i].Priority
			bestIdx = i
		}
	}
	cmd := q.commands[bestIdx]
	q.commands = append(q.commands[:bestIdx], q.commands[bestIdx+1:]...)
	return &cmd
}

func DequeueCommand() *QueuedCommand { return defaultQueue.DequeueCommand() }

// DrainCommandQueue removes and returns all commands in priority order.
func (q *Queue) DrainCommandQueue() []QueuedCommand {
	var out []QueuedCommand
	for {
		cmd := q.DequeueCommand()
		if cmd == nil {
			break
		}
		out = append(out, *cmd)
	}
	return out
}

func DrainCommandQueue() []QueuedCommand { return defaultQueue.DrainCommandQueue() }

// HasPendingNotifications returns true if any task-notification commands are queued.
func (q *Queue) HasPendingNotifications() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, c := range q.commands {
		if c.Mode == "task-notification" {
			return true
		}
	}
	return false
}

func HasPendingNotifications() bool { return defaultQueue.HasPendingNotifications() }

// ClearCommandQueue removes all queued commands.
func (q *Queue) ClearCommandQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.commands = nil
}

func ClearCommandQueue() { defaultQueue.ClearCommandQueue() }

// AddPendingBgAgent increments the pending background agent counter.
func (q *Queue) AddPendingBgAgent() { q.mu.Lock(); q.pendingBgAgents++; q.mu.Unlock() }

func AddPendingBgAgent() { defaultQueue.AddPendingBgAgent() }

// RemovePendingBgAgent decrements the pending background agent counter.
func (q *Queue) RemovePendingBgAgent() {
	q.mu.Lock()
	if q.pendingBgAgents > 0 {
		q.pendingBgAgents--
	}
	q.mu.Unlock()
}

func RemovePendingBgAgent() { defaultQueue.RemovePendingBgAgent() }

// HasPendingBgAgents returns true if background agents are still running.
func (q *Queue) HasPendingBgAgents() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.pendingBgAgents > 0
}

func HasPendingBgAgents() bool { return defaultQueue.HasPendingBgAgents() }

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

// (Queue 方法版,WS per-session 使用)
func (q *Queue) EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64) {
	xml := BuildAgentNotification(agentID, toolUseID, outputFile, status, summary, output, tokenCount, toolUseCount, durationMs)
	q.EnqueuePendingNotification(xml)
	q.RemovePendingBgAgent()
}

// (包级转发,stdio/默认单例,向后兼容)
func EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64) {
	defaultQueue.EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output, tokenCount, toolUseCount, durationMs)
}
