package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"image/color"

	"charm.land/lipgloss/v2"
	"goc/commands"
)

// taskListEntry mirrors tools.v2Task for display (decoupled from tools package).
type taskListEntry struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Status    string `json:"status"`
	BlockedBy []string `json:"blockedBy"`
}

// taskListModel manages reading and rendering a task list from disk.
type taskListModel struct {
	mu           sync.Mutex
	tasks        []taskListEntry
	completedAt  map[string]time.Time // task ID → when it transitioned to completed
	lastSnapshot map[string]string    // task ID → last known status
	hideUntil    time.Time            // hide all-tasks-completed banner until
	visible      bool
	pollTick     time.Duration
}

const (
	taskIconCompleted    = "✓"
	taskIconInProgress   = "■"
	taskIconPending      = "□"
	taskBlockedIndicator = "›"

	recentCompletedTTL    = 30 * time.Second
	taskHideAfterComplete = 5 * time.Second
	defaultPollInterval   = 2 * time.Second
)

func newTaskListModel() *taskListModel {
	return &taskListModel{
		completedAt:  make(map[string]time.Time),
		lastSnapshot: make(map[string]string),
		pollTick:     defaultPollInterval,
	}
}

// taskListID returns the task list directory name, matching TS getTaskListId logic.
func taskListID() string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TASK_LIST_ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_NAME")); v != "" {
		return v
	}
	// Fall back to session-based or default
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME")); v != "" {
		return v
	}
	return "default"
}

func tasksDir() string {
	base := commands.ClaudeConfigHome()
	id := taskListID()
	// Sanitize: keep alphanumeric, underscore, hyphen
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '-'
	}, id)
	return filepath.Join(base, "tasks", sanitized)
}

// readTasksFromDisk reads all task JSON files from the task directory.
// Returns nil if the directory doesn't exist or can't be read.
func readTasksFromDisk() []taskListEntry {
	dir := tasksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var tasks []taskListEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// Skip lock and high water mark files
		if e.Name() == ".lock" || e.Name() == ".highwatermark" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var t taskListEntry
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		if t.ID == "" || t.Subject == "" {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	return tasks
}

// poll fetches tasks from disk and updates internal state.
// Returns true if visibility changed (caller should trigger rerender).
func (tl *taskListModel) poll() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	tasks := readTasksFromDisk()

	if len(tasks) == 0 {
		// No tasks: keep invisible
		tl.visible = false
		tl.tasks = nil
		return true
	}

	// Track completion transitions
	for _, t := range tasks {
		prevStatus, seen := tl.lastSnapshot[t.ID]
		if t.Status == "completed" {
			if !seen || prevStatus != "completed" {
				tl.completedAt[t.ID] = now
			}
		} else {
			delete(tl.completedAt, t.ID)
		}
		tl.lastSnapshot[t.ID] = t.Status
	}

	// Clean up stale completion timestamps
	for id, ts := range tl.completedAt {
		if now.Sub(ts) > recentCompletedTTL {
			delete(tl.completedAt, id)
		}
		// Also remove if task is no longer in the list
		found := false
		for _, t := range tasks {
			if t.ID == id {
				found = true
				break
			}
		}
		if !found {
			delete(tl.completedAt, id)
		}
	}

	// Auto-hide when all tasks completed for >= hideAfterComplete
	allDone := len(tasks) > 0
	for _, t := range tasks {
		if t.Status != "completed" {
			allDone = false
			break
		}
	}
	if allDone {
		if tl.hideUntil.IsZero() {
			tl.hideUntil = now.Add(taskHideAfterComplete)
			tl.visible = true
		} else if now.After(tl.hideUntil) {
			tl.visible = false
			tl.tasks = tasks
			return true
		}
	} else {
		tl.hideUntil = time.Time{}
		tl.visible = true
	}

	tl.tasks = tasks
	return true
}

// isVisible reports whether the task list should be rendered.
func (tl *taskListModel) isVisible() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.visible
}

// view renders the task list into a string.
// Returns empty string if not visible.
func (tl *taskListModel) view(maxDisplay int, columns int) string {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.visible || len(tl.tasks) == 0 {
		return ""
	}

	tasks := tl.tasks
	now := time.Now()

	// Build unresolved IDs set (non-completed)
	unresolvedIDs := make(map[string]bool)
	for _, t := range tasks {
		if t.Status != "completed" {
			unresolvedIDs[t.ID] = true
		}
	}

	// Priority ordering: recent completed → in_progress → pending → older completed
	needsTruncation := len(tasks) > maxDisplay

	var visible, hidden []taskListEntry

	if needsTruncation {
		var recentCompleted, olderCompleted, inProgress, pending []taskListEntry
		for _, t := range tasks {
			switch t.Status {
			case "completed":
				ct, isRecent := tl.completedAt[t.ID]
				if isRecent && now.Sub(ct) < recentCompletedTTL {
					recentCompleted = append(recentCompleted, t)
				} else {
					olderCompleted = append(olderCompleted, t)
				}
			case "in_progress":
				inProgress = append(inProgress, t)
			default: // pending
				pending = append(pending, t)
			}
		}

		// Sort each group by ID
		sort.Slice(recentCompleted, func(i, j int) bool { return recentCompleted[i].ID < recentCompleted[j].ID })
		sort.Slice(inProgress, func(i, j int) bool { return inProgress[i].ID < inProgress[j].ID })
		sort.Slice(pending, func(i, j int) bool {
			aBlocked := false
			bBlocked := false
			for _, bid := range pending[i].BlockedBy {
				if unresolvedIDs[bid] {
					aBlocked = true
					break
				}
			}
			for _, bid := range pending[j].BlockedBy {
				if unresolvedIDs[bid] {
					bBlocked = true
					break
				}
			}
			if aBlocked != bBlocked {
				return bBlocked // unblocked first
			}
			return pending[i].ID < pending[j].ID
		})
		sort.Slice(olderCompleted, func(i, j int) bool { return olderCompleted[i].ID < olderCompleted[j].ID })

		prioritized := append(append(append(recentCompleted, inProgress...), pending...), olderCompleted...)
		if maxDisplay > 0 && maxDisplay < len(prioritized) {
			visible = prioritized[:maxDisplay]
			hidden = prioritized[maxDisplay:]
		} else {
			visible = prioritized
		}
	} else {
		visible = tasks
	}

	// Count stats
	completedCount := 0
	inProgressCount := 0
	pendingCount := 0
	for _, t := range tasks {
		switch t.Status {
		case "completed":
			completedCount++
		case "in_progress":
			inProgressCount++
		default:
			pendingCount++
		}
	}

	var b strings.Builder

	// Render each visible task
	for _, t := range visible {
		var icon string
		var color color.Color
		isBold := false
		isStrike := false
		isDim := false

		switch t.Status {
		case "completed":
			icon = taskIconCompleted
			color = lipgloss.Color("42") // green
			isStrike = true
			isDim = true
		case "in_progress":
			icon = taskIconInProgress
			color = lipgloss.Color("141") // claude purple
			isBold = true
		default: // pending
			icon = taskIconPending
			isDim = true
		}

		// Check if blocked
		var openBlockers []string
		for _, bid := range t.BlockedBy {
			if unresolvedIDs[bid] {
				openBlockers = append(openBlockers, bid)
			}
		}
		isBlocked := len(openBlockers) > 0 && t.Status != "completed"

		// Truncate subject to fit terminal width
		avail := columns - 20 // space for icon, gutter, blocking info
		if avail < 15 {
			avail = 15
		}
		subject := t.Subject
		if len(subject) > avail {
			subject = subject[:avail-1] + "…"
		}

		// Build icon
		iconStyle := lipgloss.NewStyle().Foreground(color)
		b.WriteString(iconStyle.Render(icon + " "))

		// Build subject
		subjStyle := lipgloss.NewStyle()
		if isBold {
			subjStyle = subjStyle.Bold(true)
		}
		if isStrike {
			subjStyle = subjStyle.Strikethrough(true)
		}
		if isDim {
			subjStyle = subjStyle.Faint(true)
		}
		b.WriteString(subjStyle.Render(subject))

		// Blocked indicator
		if isBlocked {
			sort.Strings(openBlockers)
			blockedIDs := make([]string, len(openBlockers))
			for i, bid := range openBlockers {
				blockedIDs[i] = "#" + bid
			}
			blockedStr := " " + taskBlockedIndicator + " blocked by " + strings.Join(blockedIDs, ", ")
			b.WriteString(lipgloss.NewStyle().Faint(true).Render(blockedStr))
		}

		b.WriteByte('\n')
	}

	// Hidden tasks summary
	if len(hidden) > 0 {
		hiddenPending := 0
		hiddenInProg := 0
		hiddenCompleted := 0
		for _, t := range hidden {
			switch t.Status {
			case "completed":
				hiddenCompleted++
			case "in_progress":
				hiddenInProg++
			default:
				hiddenPending++
			}
		}
		parts := make([]string, 0, 3)
		if hiddenInProg > 0 {
			parts = append(parts, fmt.Sprintf("%d in progress", hiddenInProg))
		}
		if hiddenPending > 0 {
			parts = append(parts, fmt.Sprintf("%d pending", hiddenPending))
		}
		if hiddenCompleted > 0 {
			parts = append(parts, fmt.Sprintf("%d completed", hiddenCompleted))
		}
		if len(parts) > 0 {
			b.WriteString(lipgloss.NewStyle().Faint(true).Render(" … +" + strings.Join(parts, ", ")))
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// taskListSummary returns a one-line task count for the header.
func (tl *taskListModel) taskCountSummary() string {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.tasks) == 0 {
		return ""
	}

	total := len(tl.tasks)
	completed := 0
	inProg := 0
	pending := 0
	for _, t := range tl.tasks {
		switch t.Status {
		case "completed":
			completed++
		case "in_progress":
			inProg++
		default:
			pending++
		}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d done", completed))
	if inProg > 0 {
		parts = append(parts, fmt.Sprintf("%d in progress", inProg))
	}
	parts = append(parts, fmt.Sprintf("%d open", pending))

	return fmt.Sprintf("%d tasks (%s)", total, strings.Join(parts, ", "))
}

// taskListTickMsg is sent by the polling tick timer.
type taskListTickMsg struct{}

// taskListTickCmd returns a command that triggers on the poll interval.
func taskListTickCmd(tl *taskListModel) tea.Cmd {
	return tea.Tick(tl.pollTick, func(time.Time) tea.Msg {
		return taskListTickMsg{}
	})
}
