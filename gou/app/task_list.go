package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"image/color"

	"charm.land/lipgloss/v2"
	"goc/tools"
)

// agentColorMap maps agent definition color names to lipgloss terminal colors.
// TS AGENT_COLOR_TO_THEME_COLOR in agentColorManager.ts.
var agentColorMap = map[string]color.Color{
	"red":     lipgloss.Color("1"),
	"blue":    lipgloss.Color("4"),
	"green":   lipgloss.Color("2"),
	"yellow":  lipgloss.Color("3"),
	"magenta": lipgloss.Color("5"),
	"cyan":    lipgloss.Color("6"),
	"orange":  lipgloss.Color("214"),
	"claude":  lipgloss.Color("141"),
}

// taskListEntry mirrors tools.v2Task for display (decoupled from full validation).
type taskListEntry struct {
	ID        string         `json:"id"`
	Subject   string         `json:"subject"`
	Status    string         `json:"status"`
	Owner     string         `json:"owner,omitempty"`
	BlockedBy []string       `json:"blockedBy"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// taskListModel manages reading and rendering a task list from disk.
type taskListModel struct {
	mu           sync.Mutex
	toolCfg      tools.Config // SessionID (and future fields) for [tools.TaskListID] / tool parity
	tasks        []taskListEntry
	completedAt  map[string]time.Time // task ID → when it transitioned to completed
	lastSnapshot map[string]string    // task ID → last known status
	hideUntil    time.Time            // hide all-tasks-completed banner until
	visible      bool
	pollTick     time.Duration
	agentTasks   *agentTaskStore // for teammate activity + owner color lookup
	tasksDir     string          // cached tasks directory path for dir watcher
	dirWatchMod  time.Time       // last observed ModTime of tasks dir
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

func newTaskListModel(sessionID string) *taskListModel {
	tl := &taskListModel{
		toolCfg:      tools.Config{SessionID: strings.TrimSpace(sessionID)},
		completedAt:  make(map[string]time.Time),
		lastSnapshot: make(map[string]string),
		pollTick:     defaultPollInterval,
	}
	tl.tasksDir = tools.V2TasksDir(tools.TaskListID(tl.toolCfg))
	return tl
}

// setAgentTasks wires the agent task store for teammate activity lookup and owner color.
func (tl *taskListModel) setAgentTasks(s *agentTaskStore) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.agentTasks = s
}

// dirChanged checks if the tasks directory mod time has changed since last check.
// Used as a lightweight file watcher without fsnotify dependency.
func (tl *taskListModel) dirChanged() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	info, err := os.Stat(tl.tasksDir)
	if err != nil {
		return false
	}
	mt := info.ModTime()
	if mt.After(tl.dirWatchMod) {
		tl.dirWatchMod = mt
		return true
	}
	return false
}

// taskMetadataInternal matches TS listTasks filter: exclude tasks with metadata._internal.
func taskMetadataInternal(meta map[string]any) bool {
	if meta == nil {
		return false
	}
	v, ok := meta["_internal"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s != "" && s != "false" && s != "0"
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return v != nil
	}
}

// poll fetches tasks from disk and updates internal state.
// Returns true if visibility changed (caller should trigger rerender).
func (tl *taskListModel) poll() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	tasks := tl.readTasksFromDiskUnlocked()

	if len(tasks) == 0 {
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

// readTasksFromDiskUnlocked is like readTasksFromDisk but assumes tl.mu is held.
func (tl *taskListModel) readTasksFromDiskUnlocked() []taskListEntry {
	id := tools.TaskListID(tl.toolCfg)
	dir := tools.V2TasksDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var tasks []taskListEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
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
		if taskMetadataInternal(t.Metadata) {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	return tasks
}

// isVisible reports whether the task list should be rendered.
func (tl *taskListModel) isVisible() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.visible
}

// prioritizeTasks returns tasks ordered for display when space is limited (caller must hold tl.mu for completedAt).
func (tl *taskListModel) prioritizeTasks(tasks []taskListEntry, now time.Time, unresolvedIDs map[string]bool) []taskListEntry {
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
		default:
			pending = append(pending, t)
		}
	}

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
			return bBlocked
		}
		return pending[i].ID < pending[j].ID
	})
	sort.Slice(olderCompleted, func(i, j int) bool { return olderCompleted[i].ID < olderCompleted[j].ID })

	return append(append(append(recentCompleted, inProgress...), pending...), olderCompleted...)
}

// taskListStandaloneHeader matches TS TaskListV2 isStandalone summary line: "N tasks (X done, [Y in progress, ]Z open)".
func taskListStandaloneHeader(tasks []taskListEntry) string {
	if len(tasks) == 0 {
		return ""
	}
	total := len(tasks)
	completed := 0
	inProgress := 0
	pending := 0
	for _, t := range tasks {
		switch t.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		default:
			pending++
		}
	}
	faint := lipgloss.NewStyle().Faint(true)
	num := lipgloss.NewStyle().Faint(true).Bold(true)
	var b strings.Builder
	b.WriteString(num.Render(strconv.Itoa(total)))
	b.WriteString(faint.Render(" tasks ("))
	b.WriteString(num.Render(strconv.Itoa(completed)))
	b.WriteString(faint.Render(" done, "))
	if inProgress > 0 {
		b.WriteString(num.Render(strconv.Itoa(inProgress)))
		b.WriteString(faint.Render(" in progress, "))
	}
	b.WriteString(num.Render(strconv.Itoa(pending)))
	b.WriteString(faint.Render(" open)"))
	return b.String()
}

// view renders the task list into a string (isStandalone-style: summary line + rows).
// Returns empty string if not visible.
func (tl *taskListModel) view(maxDisplay int, columns int) string {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	return tl.viewUnlocked(maxDisplay, columns)
}

// viewUnlocked is the internal view renderer (caller holds tl.mu).
func (tl *taskListModel) viewUnlocked(maxDisplay int, columns int) string {
	if !tl.visible || len(tl.tasks) == 0 {
		return ""
	}

	tasks := tl.tasks
	now := time.Now()

	unresolvedIDs := make(map[string]bool)
	for _, t := range tasks {
		if t.Status != "completed" {
			unresolvedIDs[t.ID] = true
		}
	}

	needsTruncation := maxDisplay >= 0 && len(tasks) > maxDisplay
	if maxDisplay < 0 {
		maxDisplay = 0
	}

	var visible, hidden []taskListEntry

	if needsTruncation {
		pri := tl.prioritizeTasks(tasks, now, unresolvedIDs)
		if maxDisplay == 0 {
			visible = nil
			hidden = pri
		} else {
			if maxDisplay < len(pri) {
				visible = pri[:maxDisplay]
				hidden = pri[maxDisplay:]
			} else {
				visible = pri
			}
		}
	} else {
		visible = tasks
	}

	// Build activity lookup from agentTasks: owner name → current summary/activity
	ownerActivity := make(map[string]string)
	ownerActive := make(map[string]bool)
	ownerColorName := make(map[string]string)
	if tl.agentTasks != nil {
		for _, at := range tl.agentTasks.VisibleTasks() {
			if at.Status == "running" {
				ownerActive[at.Name] = true
				ownerActive[at.AgentType] = true
				if at.Progress != nil {
					act := at.Progress.Summary
					if act == "" {
						act = at.Progress.LastActivityDesc
					}
					if act != "" {
						ownerActivity[at.Name] = act
						ownerActivity[at.AgentType] = act
					}
				}
				// Look up agent color from definitions
				if colorName := resolveAgentColorName(at.AgentType); colorName != "" {
					ownerColorName[at.Name] = colorName
					ownerColorName[at.AgentType] = colorName
				}
			}
		}
	}

	var b strings.Builder
	if header := taskListStandaloneHeader(tasks); header != "" {
		b.WriteString(header)
		b.WriteByte('\n')
	}

	for _, t := range visible {
		var icon string
		var c color.Color
		isBold := false
		isStrike := false
		isDim := false

		switch t.Status {
		case "completed":
			icon = taskIconCompleted
			c = lipgloss.Color("42")
			isStrike = true
			isDim = true
		case "in_progress":
			icon = taskIconInProgress
			c = lipgloss.Color("141")
			isBold = true
		default:
			icon = taskIconPending
			isDim = true
		}

		var openBlockers []string
		for _, bid := range t.BlockedBy {
			if unresolvedIDs[bid] {
				openBlockers = append(openBlockers, bid)
			}
		}
		isBlocked := len(openBlockers) > 0 && t.Status != "completed"

		avail := columns - 20
		if avail < 15 {
			avail = 15
		}
		subject := t.Subject
		if len(subject) > avail {
			subject = subject[:avail-1] + "…"
		}

		iconStyle := lipgloss.NewStyle().Foreground(c)
		b.WriteString(iconStyle.Render(icon + " "))

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

		// Show owner for non-completed tasks (mirrors TS TaskListV2 owner display)
		if t.Owner != "" && t.Status != "completed" && columns >= 60 && ownerActive[t.Owner] {
			ownerStyle := lipgloss.NewStyle().Faint(true)
			if cn, ok := ownerColorName[t.Owner]; ok {
				if oc, ok2 := agentColorMap[cn]; ok2 {
					ownerStyle = lipgloss.NewStyle().Foreground(oc)
				}
			}
			ownerStr := " (@" + t.Owner + ")"
			b.WriteString(ownerStyle.Render(ownerStr))
		} else if t.Owner != "" && t.Status != "completed" && columns >= 60 {
			b.WriteString(lipgloss.NewStyle().Faint(true).Render(" (@" + t.Owner + ")"))
		}

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

		// Show teammate activity on second line for in-progress, non-blocked, owned tasks
		showActivity := t.Status == "in_progress" && !isBlocked && t.Owner != ""
		if showActivity {
			act := ownerActivity[t.Owner]
			if act != "" {
				maxActivityWidth := columns - 15
				if maxActivityWidth < 15 {
					maxActivityWidth = 15
				}
				if len(act) > maxActivityWidth {
					act = act[:maxActivityWidth-1] + "…"
				}
				b.WriteString("  ")
				b.WriteString(lipgloss.NewStyle().Faint(true).Render(act + "…"))
				b.WriteByte('\n')
			}
		}
	}

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

// resolveAgentColorName returns the color name from an agent definition matching the given agentType.
func resolveAgentColorName(agentType string) string {
	defs := tools.LoadAgentDefinitionsForCwd("")
	for _, d := range defs {
		if strings.EqualFold(d.AgentType, agentType) {
			return strings.ToLower(strings.TrimSpace(d.Color))
		}
	}
	return ""
}

// taskListTickMsg is sent by the polling tick timer.
type taskListTickMsg struct{}

// taskListTickCmd returns a command that triggers on the poll interval.
func taskListTickCmd(tl *taskListModel) tea.Cmd {
	return tea.Tick(tl.pollTick, func(time.Time) tea.Msg {
		return taskListTickMsg{}
	})
}

// taskListTickCmdAgent returns a command that fires a 1s tick for agent coordinator panel refresh.
func taskListTickCmdAgent() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return AgentTaskTickMsg{}
	})
}
