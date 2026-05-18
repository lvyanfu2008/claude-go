package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"goc/commands"
	"goc/hookexec"
)

// TS parity reference: claude-code/src/utils/tasks.ts (layout, locks, high water mark, CRUD).

// TaskType constants mirror TS task type definitions (src/tasks/taskDefinitions.ts).
const (
	TaskTypeLocalBash          = "local_bash"
	TaskTypeLocalAgent         = "local_agent"
	TaskTypeRemoteAgent        = "remote_agent"
	TaskTypeInProcessTeammate  = "in_process_teammate"
	TaskTypeLocalWorkflow      = "local_workflow"
	TaskTypeMonitorMCP         = "monitor_mcp"
	TaskTypeDream              = "dream"
)

// taskTypeToPrefix maps task types to single-character ID prefixes (TS: TASK_TYPE_TO_PREFIX).
var taskTypeToPrefix = map[string]string{
	TaskTypeLocalBash:         "b",
	TaskTypeLocalAgent:        "a",
	TaskTypeRemoteAgent:       "r",
	TaskTypeInProcessTeammate: "t",
	TaskTypeLocalWorkflow:     "w",
	TaskTypeMonitorMCP:        "m",
	TaskTypeDream:             "d",
}

// prefixToTaskType maps ID prefixes back to task types.
var prefixToTaskType = map[string]string{
	"b": TaskTypeLocalBash,
	"a": TaskTypeLocalAgent,
	"r": TaskTypeRemoteAgent,
	"t": TaskTypeInProcessTeammate,
	"w": TaskTypeLocalWorkflow,
	"m": TaskTypeMonitorMCP,
	"d": TaskTypeDream,
}

// validTaskTypes is the set of all valid task type strings.
var validTaskTypes = map[string]bool{
	TaskTypeLocalBash:         true,
	TaskTypeLocalAgent:        true,
	TaskTypeRemoteAgent:       true,
	TaskTypeInProcessTeammate: true,
	TaskTypeLocalWorkflow:     true,
	TaskTypeMonitorMCP:        true,
	TaskTypeDream:             true,
}

// Task status constants (extended with failed/killed for state machine).
const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusKilled     = "killed"
)

// terminalTaskStatus is the set of statuses from which no further transitions are allowed.
var terminalTaskStatus = map[string]bool{
	TaskStatusCompleted: true,
	TaskStatusFailed:    true,
	TaskStatusKilled:    true,
}

// validTaskStatus is the set of all valid task statuses.
var validTaskStatus = map[string]bool{
	TaskStatusPending:    true,
	TaskStatusInProgress: true,
	TaskStatusCompleted:  true,
	TaskStatusFailed:     true,
	TaskStatusKilled:     true,
}

// isValidTransition checks whether a status transition is valid.
// Transitions: pending → in_progress → completed/failed/killed.
// Same-status transitions are idempotent (allowed).
// Terminal statuses (completed/failed/killed) cannot be transitioned from.
func isValidTransition(from, to string) bool {
	if from == to {
		return true
	}
	if terminalTaskStatus[from] {
		return false
	}
	if from == TaskStatusPending && to == TaskStatusInProgress {
		return true
	}
	if from == TaskStatusInProgress && terminalTaskStatus[to] {
		return true
	}
	return false
}

const (
	v2HighWaterMarkFile = ".highwatermark"
	v2LockRetries       = 30
	v2LockMinBackoff    = 5 * time.Millisecond
	v2LockMaxBackoff    = 100 * time.Millisecond
)

var v2PathSanitize = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizePathComponentV2(s string) string {
	return v2PathSanitize.ReplaceAllString(s, "-")
}

// TaskListID resolves the task list directory name (getTaskListId in TS). Used by v2 task tools and gou-demo TUI.
func TaskListID(cfg Config) string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TASK_LIST_ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_NAME")); v != "" {
		return v
	}
	// In-process teammate: resolve team name from agent identity in team roster
	if agentName := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME")); agentName != "" {
		if teamName, _, _ := findTeamMemberByName(agentName); teamName != "" {
			return teamName
		}
	}
	if v := strings.TrimSpace(cfg.SessionID); v != "" {
		return v
	}
	return "default"
}

func V2TasksDir(taskListID string) string {
	base := commands.ClaudeConfigHome()
	id := sanitizePathComponentV2(taskListID)
	return filepath.Join(base, "tasks", id)
}

func v2TaskPath(taskListID, taskID string) string {
	return filepath.Join(V2TasksDir(taskListID), sanitizePathComponentV2(taskID)+".json")
}

func v2ListLockPath(taskListID string) string {
	return filepath.Join(V2TasksDir(taskListID), ".lock")
}

func v2HighWaterMarkPath(taskListID string) string {
	return filepath.Join(V2TasksDir(taskListID), v2HighWaterMarkFile)
}

func ensureEmptyLockFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	return f.Close()
}

// withListExclusiveLock locks the task list `.lock` file (must call ensureEmptyLockFile first).
func withListExclusiveLock(lockPath string, fn func() error) (err error) {
	if err := ensureEmptyLockFile(lockPath); err != nil {
		return err
	}
	fl := flock.New(lockPath)
	for attempt := 0; attempt < v2LockRetries; attempt++ {
		locked, err := fl.TryLock()
		if err != nil {
			return err
		}
		if locked {
			defer func() { _ = fl.Unlock() }()
			return fn()
		}
		shift := attempt
		if shift > 10 {
			shift = 10
		}
		d := v2LockMinBackoff * time.Duration(1<<shift)
		if d > v2LockMaxBackoff {
			d = v2LockMaxBackoff
		}
		time.Sleep(d)
	}
	return fmt.Errorf("lock timeout: %s", lockPath)
}

// withExistingFileExclusiveLock locks an existing file (task JSON); never creates the path.
func withExistingFileExclusiveLock(path string, fn func() error) (err error) {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	fl := flock.New(path)
	for attempt := 0; attempt < v2LockRetries; attempt++ {
		locked, err := fl.TryLock()
		if err != nil {
			return err
		}
		if locked {
			defer func() { _ = fl.Unlock() }()
			return fn()
		}
		shift := attempt
		if shift > 10 {
			shift = 10
		}
		d := v2LockMinBackoff * time.Duration(1<<shift)
		if d > v2LockMaxBackoff {
			d = v2LockMaxBackoff
		}
		time.Sleep(d)
	}
	return fmt.Errorf("lock timeout: %s", path)
}

func readHighWaterMarkV2(taskListID string) int {
	b, err := os.ReadFile(v2HighWaterMarkPath(taskListID))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func writeHighWaterMarkV2(taskListID string, value int) error {
	path := v2HighWaterMarkPath(taskListID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeFileAtomic(path, []byte(strconv.Itoa(value)), 0o644)
}

func findHighestTaskIDFromFilesV2(taskListID string) int {
	dir := V2TasksDir(taskListID)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	highest := 0
	for _, e := range ents {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		n, err := strconv.Atoi(base)
		if err == nil && n > highest {
			highest = n
		}
	}
	return highest
}

func findHighestTaskIDV2(taskListID string) int {
	a := findHighestTaskIDFromFilesV2(taskListID)
	b := readHighWaterMarkV2(taskListID)
	if a > b {
		return a
	}
	return b
}

type v2Task struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	ActiveForm  string         `json:"activeForm,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	Status      string         `json:"status"`
	Blocks      []string       `json:"blocks"`
	BlockedBy   []string       `json:"blockedBy"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func validateV2Task(t *v2Task) bool {
	if t == nil {
		return false
	}
	if !validTaskStatus[t.Status] {
		return false
	}
	// Type is optional when reading legacy tasks, but must be valid if present.
	if t.Type != "" && !validTaskTypes[t.Type] {
		return false
	}
	if strings.TrimSpace(t.ID) == "" || strings.TrimSpace(t.Subject) == "" {
		return false
	}
	if t.Blocks == nil || t.BlockedBy == nil {
		return false
	}
	return true
}

func v2GetTask(taskListID, taskID string) (*v2Task, error) {
	path := v2TaskPath(taskListID, taskID)
	b, err := readFileLimited(path, 1<<20)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var t v2Task
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, nil
	}
	if !validateV2Task(&t) {
		return nil, nil
	}
	return &t, nil
}

func v2WriteTask(taskListID string, t *v2Task) error {
	if t.Blocks == nil {
		t.Blocks = []string{}
	}
	if t.BlockedBy == nil {
		t.BlockedBy = []string{}
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeFileAtomic(v2TaskPath(taskListID, t.ID), b, 0o644)
}

func v2CreateTask(taskListID, taskType, subject, description, activeForm string, metadata map[string]any) (string, error) {
	return v2CreateTaskFull(taskListID, taskType, subject, description, activeForm, "", "", nil, nil, metadata)
}

func v2CreateTaskFull(taskListID, taskType, subject, description, activeForm, status, owner string, blocks, blockedBy []string, metadata map[string]any) (string, error) {
	if taskType == "" {
		taskType = TaskTypeLocalAgent
	}
	if !validTaskTypes[taskType] {
		return "", fmt.Errorf("invalid task type: %q", taskType)
	}
	if status != "" && !validTaskStatus[status] {
		return "", fmt.Errorf("invalid status: %q", status)
	}
	if status == "" {
		status = "pending"
	}
	lockPath := v2ListLockPath(taskListID)
	var newID string
	err := withListExclusiveLock(lockPath, func() error {
		highest := findHighestTaskIDV2(taskListID)
		newID = strconv.Itoa(highest + 1)
		md := metadata
		if len(md) == 0 {
			md = nil
		}
		if owner == "" && commands.AgentSwarmsEnabled() {
			owner = strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME"))
			if owner == "" {
				owner = strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_ID"))
			}
		}
		b := blocks
		if b == nil {
			b = []string{}
		}
		bb := blockedBy
		if bb == nil {
			bb = []string{}
		}
		t := &v2Task{
			ID:          newID,
			Type:        taskType,
			Subject:     subject,
			Description: description,
			ActiveForm:  activeForm,
			Owner:       owner,
			Status:      status,
			Blocks:      b,
			BlockedBy:   bb,
			Metadata:    md,
		}
		return v2WriteTask(taskListID, t)
	})
	if err != nil {
		return "", err
	}
	return newID, nil
}

func v2ListTasks(taskListID string) ([]*v2Task, error) {
	dir := V2TasksDir(taskListID)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasSuffix(name, ".json") && !strings.HasPrefix(name, ".") {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(ids)
	var out []*v2Task
	for _, id := range ids {
		t, err := v2GetTask(taskListID, id)
		if err != nil {
			return nil, err
		}
		if t != nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func v2UpdateTaskUnsafe(taskListID, taskID string, patch *v2Task) error {
	return v2WriteTask(taskListID, patch)
}

func v2UpdateTaskFields(taskListID, taskID string, updates map[string]any) (*v2Task, error) {
	path := v2TaskPath(taskListID, taskID)
	existing, err := v2GetTask(taskListID, taskID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	var merged *v2Task
	err = withExistingFileExclusiveLock(path, func() error {
		cur, err := v2GetTask(taskListID, taskID)
		if err != nil {
			return err
		}
		if cur == nil {
			return fmt.Errorf("task missing under lock")
		}
		next := *cur
		if v, ok := updates["subject"]; ok {
			next.Subject, _ = v.(string)
		}
		if v, ok := updates["description"]; ok {
			next.Description, _ = v.(string)
		}
		if v, ok := updates["activeForm"]; ok {
			next.ActiveForm, _ = v.(string)
		}
		if v, ok := updates["owner"]; ok {
			next.Owner, _ = v.(string)
		}
		if v, ok := updates["status"]; ok {
			next.Status, _ = v.(string)
		}
		if v, ok := updates["blocks"]; ok {
			if sl, ok := v.([]string); ok {
				next.Blocks = sl
			}
		}
		if v, ok := updates["blockedBy"]; ok {
			if sl, ok := v.([]string); ok {
				next.BlockedBy = sl
			}
		}
		if v, ok := updates["metadata"]; ok {
			if m, ok := v.(map[string]any); ok {
				next.Metadata = m
			}
		}
		if !validateV2Task(&next) {
			return fmt.Errorf("invalid task after patch")
		}
		merged = &next
		return v2UpdateTaskUnsafe(taskListID, taskID, merged)
	})
	if err != nil {
		return nil, err
	}
	return merged, nil
}

func v2DeleteTask(taskListID, taskID string) (bool, error) {
	path := v2TaskPath(taskListID, taskID)
	if n, err := strconv.Atoi(taskID); err == nil && n > 0 {
		cur := readHighWaterMarkV2(taskListID)
		if n > cur {
			if err := writeHighWaterMarkV2(taskListID, n); err != nil {
				return false, err
			}
		}
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	all, err := v2ListTasks(taskListID)
	if err != nil {
		return false, err
	}
	for _, task := range all {
		newB := filterID(task.Blocks, taskID)
		newBB := filterID(task.BlockedBy, taskID)
		if len(newB) != len(task.Blocks) || len(newBB) != len(task.BlockedBy) {
			_, err := v2UpdateTaskFields(taskListID, task.ID, map[string]any{
				"blocks":    newB,
				"blockedBy": newBB,
			})
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func filterID(ids []string, remove string) []string {
	var out []string
	for _, id := range ids {
		if id != remove {
			out = append(out, id)
		}
	}
	return out
}

func v2BlockTask(taskListID, fromTaskID, toTaskID string) error {
	from, err := v2GetTask(taskListID, fromTaskID)
	if err != nil {
		return err
	}
	to, err := v2GetTask(taskListID, toTaskID)
	if err != nil {
		return err
	}
	if from == nil || to == nil {
		return fmt.Errorf("blockTask: missing task")
	}
	if !containsID(from.Blocks, toTaskID) {
		b := append(append([]string(nil), from.Blocks...), toTaskID)
		if _, err := v2UpdateTaskFields(taskListID, fromTaskID, map[string]any{"blocks": b}); err != nil {
			return err
		}
	}
	if !containsID(to.BlockedBy, fromTaskID) {
		bb := append(append([]string(nil), to.BlockedBy...), fromTaskID)
		if _, err := v2UpdateTaskFields(taskListID, toTaskID, map[string]any{"blockedBy": bb}); err != nil {
			return err
		}
	}
	return nil
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func metadataInternalTruthy(m map[string]any) bool {
	v, ok := m["_internal"]
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return strings.TrimSpace(x) != ""
	default:
		return true
	}
}

func errTodoV2Disabled(tool string) error {
	return fmt.Errorf("%s: Todo v2 tools disabled (non-interactive). Set CLAUDE_CODE_ENABLE_TASKS=1 to enable", tool)
}

// broadcastTaskEvent sends a task-related notification to all team members except self.
func broadcastTaskEvent(taskID, subject, event string) {
	agentName := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME"))
	agentID := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_ID"))
	teamName := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_NAME"))
	if teamName == "" {
		return
	}
	sender := agentName
	if sender == "" {
		sender = agentID
	}
	if sender == "" {
		return
	}
	tf, err := readTeamFile(teamName)
	if err != nil || tf == nil {
		return
	}
	msg := fmt.Sprintf("<task-notification>\n  <event>%s</event>\n  <task-id>%s</task-id>\n  <subject>%s</subject>\n</task-notification>", event, taskID, xmlEscape(subject))
	for _, m := range tf.Members {
		if m.AgentID == agentID || m.Name == agentName {
			continue
		}
		targetName := m.Name
		if targetName == "" {
			targetName = m.AgentID
		}
		_ = writeToMailbox(targetName, teamName, sender, msg)
	}
}

// TaskCreateFromJSON implements TaskCreate (TS TaskCreateTool) with synchronous hook execution.
func TaskCreateFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskCreate")
	}
	var in struct {
		Type        string         `json:"type"`
		Subject     string         `json:"subject"`
		Description string         `json:"description"`
		ActiveForm  string         `json:"activeForm"`
		Status      string         `json:"status"`
		Owner       string         `json:"owner"`
		Blocks      []string       `json:"blocks"`
		BlockedBy   []string       `json:"blockedBy"`
		Metadata    map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Subject) == "" {
		return "", true, fmt.Errorf("subject is required")
	}
	tid := TaskListID(cfg)
	id, err := v2CreateTaskFull(tid, in.Type, in.Subject, in.Description, in.ActiveForm, in.Status, in.Owner, in.Blocks, in.BlockedBy, in.Metadata)
	if err != nil {
		return "", true, err
	}

	// Run hooks synchronously — if any block, delete the task and return error (TS: throws Error).
	blockingErrors := runTaskCreatedHook(cfg, id, tid, in.Subject, in.Description)
	if len(blockingErrors) > 0 {
		_, _ = v2DeleteTask(tid, id)
		return "", true, fmt.Errorf("%s", strings.Join(blockingErrors, "\n"))
	}

	broadcastTaskEvent(id, in.Subject, "created")
	out := map[string]any{
		"data": map[string]any{
			"task": map[string]any{"id": id, "subject": in.Subject},
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TaskGetFromJSON implements TaskGet (TS TaskGetTool).
func TaskGetFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskGet")
	}
	var in struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.TaskID) == "" {
		return "", true, fmt.Errorf("taskId is required")
	}
	t, err := v2GetTask(TaskListID(cfg), in.TaskID)
	if err != nil {
		return "", true, err
	}
	var taskPayload any
	if t == nil {
		taskPayload = nil
	} else {
		taskPayload = map[string]any{
			"id":          t.ID,
			"subject":     t.Subject,
			"description": t.Description,
			"status":      t.Status,
			"blocks":      t.Blocks,
			"blockedBy":   t.BlockedBy,
		}
	}
	out := map[string]any{"data": map[string]any{"task": taskPayload}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TaskListFromJSON implements TaskList (TS TaskListTool).
func TaskListFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	_ = raw
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskList")
	}
	all, err := v2ListTasks(TaskListID(cfg))
	if err != nil {
		return "", true, err
	}
	filtered := make([]*v2Task, 0, len(all))
	for _, t := range all {
		if metadataInternalTruthy(t.Metadata) {
			continue
		}
		filtered = append(filtered, t)
	}
	resolved := map[string]struct{}{}
	for _, t := range filtered {
		if t.Status == "completed" {
			resolved[t.ID] = struct{}{}
		}
	}
	type row struct {
		ID        string   `json:"id"`
		Subject   string   `json:"subject"`
		Status    string   `json:"status"`
		Owner     string   `json:"owner,omitempty"`
		BlockedBy []string `json:"blockedBy"`
	}
	var rows []row
	for _, t := range filtered {
		bb := make([]string, 0, len(t.BlockedBy))
		for _, id := range t.BlockedBy {
			if _, ok := resolved[id]; ok {
				continue
			}
			bb = append(bb, id)
		}
		rows = append(rows, row{
			ID:        t.ID,
			Subject:   t.Subject,
			Status:    t.Status,
			Owner:     t.Owner,
			BlockedBy: bb,
		})
	}
	out := map[string]any{"data": map[string]any{"tasks": rows}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TaskUpdateFromJSON implements TaskUpdate (TS TaskUpdateTool) with sync hooks, owner auto-assignment, mailbox notification.
func TaskUpdateFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskUpdate")
	}
	var in struct {
		TaskID       string         `json:"taskId"`
		Subject      *string        `json:"subject"`
		Description  *string        `json:"description"`
		ActiveForm   *string        `json:"activeForm"`
		Status       *string        `json:"status"`
		Owner        *string        `json:"owner"`
		AddBlocks    []string       `json:"addBlocks"`
		AddBlockedBy []string       `json:"addBlockedBy"`
		Metadata     map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.TaskID) == "" {
		return "", true, fmt.Errorf("taskId is required")
	}
	tid := TaskListID(cfg)
	existing, err := v2GetTask(tid, in.TaskID)
	if err != nil {
		return "", true, err
	}
	if existing == nil {
		out := map[string]any{
			"data": map[string]any{
				"success":       false,
				"taskId":        in.TaskID,
				"updatedFields": []string{},
				"error":         "Task not found",
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	updatedFields := []string{}
	updates := map[string]any{}

	if in.Subject != nil && *in.Subject != existing.Subject {
		updates["subject"] = *in.Subject
		updatedFields = append(updatedFields, "subject")
	}
	if in.Description != nil && *in.Description != existing.Description {
		updates["description"] = *in.Description
		updatedFields = append(updatedFields, "description")
	}
	if in.ActiveForm != nil && *in.ActiveForm != existing.ActiveForm {
		updates["activeForm"] = *in.ActiveForm
		updatedFields = append(updatedFields, "activeForm")
	}

	// Auto-set owner when transitioning to in_progress without explicit owner (TS lines 185-199).
	if in.Status != nil && *in.Status == "in_progress" && in.Owner == nil && existing.Owner == "" && commands.AgentSwarmsEnabled() {
		agentName := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME"))
		if agentName == "" {
			agentName = strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_ID"))
		}
		if agentName != "" {
			updates["owner"] = agentName
			updatedFields = append(updatedFields, "owner")
		}
	}

	if in.Owner != nil && *in.Owner != existing.Owner {
		updates["owner"] = *in.Owner
		updatedFields = append(updatedFields, "owner")
	}
	if in.Metadata != nil {
		merged := map[string]any{}
		for k, v := range existing.Metadata {
			merged[k] = v
		}
		for k, v := range in.Metadata {
			if v == nil {
				delete(merged, k)
			} else {
				merged[k] = v
			}
		}
		updates["metadata"] = merged
		updatedFields = append(updatedFields, "metadata")
	}

	var statusChange map[string]string
	if in.Status != nil {
		st := *in.Status
		if st == "deleted" {
			ok, err := v2DeleteTask(tid, in.TaskID)
			if err != nil {
				return "", true, err
			}
			data := map[string]any{
				"success":       ok,
				"taskId":        in.TaskID,
				"updatedFields": []string{},
			}
			if ok {
				data["updatedFields"] = []string{"deleted"}
				data["statusChange"] = map[string]string{"from": existing.Status, "to": "deleted"}
			} else {
				data["error"] = "Failed to delete task"
			}
			out := map[string]any{"data": data}
			b, _ := json.Marshal(out)
			return string(b), false, nil
		}
		switch st {
		case "pending", "in_progress", "completed", "failed", "killed":
		default:
			return "", true, fmt.Errorf("invalid status %q", st)
		}
		if !isValidTransition(existing.Status, st) {
			return "", true, fmt.Errorf("invalid status transition from %q to %q", existing.Status, st)
		}
		if st != existing.Status {
			if st == "completed" {
				// Block completion if any blockers are still unresolved.
				if len(existing.BlockedBy) > 0 {
					resolved := map[string]bool{}
					for _, id := range existing.BlockedBy {
						bt, err := v2GetTask(tid, id)
						if err == nil && bt != nil && bt.Status == "completed" {
							resolved[id] = true
						}
					}
					var unresolved []string
					for _, id := range existing.BlockedBy {
						if !resolved[id] {
							unresolved = append(unresolved, id)
						}
					}
					if len(unresolved) > 0 {
						return "", true, fmt.Errorf("cannot complete task: blocked by unresolved task(s): %s", strings.Join(unresolved, ", "))
					}
				}

				// Run hooks synchronously BEFORE applying the update (TS: hooks run first).
				blockingErrors := runTaskCompletedHook(cfg, in.TaskID, tid, existing.Subject, existing.Description)
				if len(blockingErrors) > 0 {
					out := map[string]any{
						"data": map[string]any{
							"success":       false,
							"taskId":        in.TaskID,
							"updatedFields": []string{},
							"error":         strings.Join(blockingErrors, "\n"),
						},
					}
					b, _ := json.Marshal(out)
					return string(b), false, nil
				}
			}

			updates["status"] = st
			updatedFields = append(updatedFields, "status")
			statusChange = map[string]string{"from": existing.Status, "to": st}
		}
	}

	if len(updates) > 0 {
		if _, err := v2UpdateTaskFields(tid, in.TaskID, updates); err != nil {
			return "", true, err
		}
	}

	// Notify new owner via mailbox when ownership changes (TS lines 276-298).
	// ownerKey tracks the effective owner: explicit input, or auto-assigned above.
	ownerKey := ""
	if v, ok := updates["owner"]; ok {
		ownerKey, _ = v.(string)
	}
	if ownerKey != "" && commands.AgentSwarmsEnabled() {
		senderName := strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_NAME"))
		if senderName == "" {
			senderName = strings.TrimSpace(os.Getenv("CLAUDE_CODE_AGENT_ID"))
		}
		if senderName == "" {
			senderName = "team-lead"
		}
		assignmentJSON, _ := json.Marshal(map[string]string{
			"type":        "task_assignment",
			"taskId":      in.TaskID,
			"subject":     existing.Subject,
			"description": existing.Description,
			"assignedBy":  senderName,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
		_ = writeToMailbox(ownerKey, tid, senderName, string(assignmentJSON))
	}

	if len(in.AddBlocks) > 0 {
		var newBlocks []string
		for _, bid := range in.AddBlocks {
			cur, err := v2GetTask(tid, in.TaskID)
			if err != nil {
				return "", true, err
			}
			if cur == nil {
				return "", true, fmt.Errorf("task disappeared during update")
			}
			if !containsID(cur.Blocks, bid) {
				if err := v2BlockTask(tid, in.TaskID, bid); err != nil {
					return "", true, err
				}
				newBlocks = append(newBlocks, bid)
			}
		}
		if len(newBlocks) > 0 {
			updatedFields = append(updatedFields, "blocks")
		}
	}
	if len(in.AddBlockedBy) > 0 {
		var added []string
		for _, blocker := range in.AddBlockedBy {
			cur, err := v2GetTask(tid, in.TaskID)
			if err != nil {
				return "", true, err
			}
			if cur == nil {
				return "", true, fmt.Errorf("task disappeared during update")
			}
			if !containsID(cur.BlockedBy, blocker) {
				if err := v2BlockTask(tid, blocker, in.TaskID); err != nil {
					return "", true, err
				}
				added = append(added, blocker)
			}
		}
		if len(added) > 0 {
			updatedFields = append(updatedFields, "blockedBy")
		}
	}

	// Broadcast events for terminal status transitions.
	if statusChange != nil {
		to := statusChange["to"]
		if to == "completed" || to == "failed" || to == "killed" {
			broadcastTaskEvent(in.TaskID, existing.Subject, to)
		}
	}

	out := map[string]any{
		"data": map[string]any{
			"success":       true,
			"taskId":        in.TaskID,
			"updatedFields": updatedFields,
		},
	}
	if statusChange != nil {
		out["data"].(map[string]any)["statusChange"] = statusChange
		if statusChange["to"] == "completed" {
			// Verification nudge: matching TS conditions (TaskUpdateTool.ts lines 333-349).
			nudge := shouldVerificationNudge(tid, cfg)
			if nudge {
				out["data"].(map[string]any)["verificationNudgeNeeded"] = true
			}
		}
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// shouldVerificationNudge mirrors TS verification nudge logic (TaskUpdateTool.ts lines 333-349).
func shouldVerificationNudge(taskListID string, cfg Config) bool {
	// In Go, we check: all tasks done, 3+ tasks, none match /verif/i.
	// We skip the TS compile-time feature('VERIFICATION_AGENT') and GrowthBook checks
	// since those are TS-specific — the tool result text guides the model regardless.
	all, err := v2ListTasks(taskListID)
	if err != nil || len(all) < 3 {
		return false
	}
	for _, t := range all {
		if t.Status != "completed" {
			return false
		}
		if strings.Contains(strings.ToLower(t.Subject), "verif") {
			return false
		}
	}
	return true
}

// runTaskCreatedHook runs TaskCreated hooks synchronously and returns blocking errors.
// Mirrors TS executeTaskCreatedHooks in hooks.ts — caller must delete task on blocking error.
func runTaskCreatedHook(cfg Config, taskID, taskListID, subject, description string) []string {
	table, err := hookexec.MergedHooksForCwd(cfg.ProjectRoot)
	if err != nil || len(table) == 0 {
		return nil
	}
	base := hookexec.BaseHookInput{
		SessionID: cfg.SessionID,
		Cwd:       cfg.WorkDir,
	}
	results, _ := hookexec.RunTaskCreatedHooks(
		context.Background(), table, cfg.WorkDir, base,
		taskID, taskListID, subject, description,
		hookexec.DefaultHookTimeoutMs,
	)
	var blockingErrors []string
	for _, r := range results {
		if r.BlockingError != nil {
			msg := "TaskCreated hook feedback:\n" + r.BlockingError.BlockingError
			blockingErrors = append(blockingErrors, msg)
		}
	}
	return blockingErrors
}

// runTaskCompletedHook runs TaskCompleted hooks synchronously and returns blocking errors.
// Mirrors TS executeTaskCompletedHooks in hooks.ts — caller must reject completion on blocking error.
func runTaskCompletedHook(cfg Config, taskID, taskListID, subject, description string) []string {
	table, err := hookexec.MergedHooksForCwd(cfg.ProjectRoot)
	if err != nil || len(table) == 0 {
		return nil
	}
	base := hookexec.BaseHookInput{
		SessionID: cfg.SessionID,
		Cwd:       cfg.WorkDir,
	}
	results, _ := hookexec.RunTaskCompletedHooks(
		context.Background(), table, cfg.WorkDir, base,
		taskID, taskListID,
		hookexec.DefaultHookTimeoutMs,
	)
	var blockingErrors []string
	for _, r := range results {
		if r.BlockingError != nil {
			msg := "TaskCompleted hook feedback:\n" + r.BlockingError.BlockingError
			blockingErrors = append(blockingErrors, msg)
		}
	}
	return blockingErrors
}

// claimTaskResult mirrors TS ClaimTaskResult.
type claimTaskResult struct {
	Success         bool     `json:"success"`
	Reason          string   `json:"reason,omitempty"`
	Task            *v2Task  `json:"task,omitempty"`
	BusyWithTasks   []string `json:"busyWithTasks,omitempty"`
	BlockedByTasks  []string `json:"blockedByTasks,omitempty"`
}

// claimTask tries to claim a task for an agent. Mirrors TS claimTask in utils/tasks.ts.
func claimTask(taskListID, taskID, claimantAgentID string) (*claimTaskResult, error) {
	existing, err := v2GetTask(taskListID, taskID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &claimTaskResult{Success: false, Reason: "task_not_found"}, nil
	}
	path := v2TaskPath(taskListID, taskID)
	var result *claimTaskResult
	err = withExistingFileExclusiveLock(path, func() error {
		cur, err := v2GetTask(taskListID, taskID)
		if err != nil {
			return err
		}
		if cur == nil {
			result = &claimTaskResult{Success: false, Reason: "task_not_found"}
			return nil
		}
		if cur.Owner != "" && cur.Owner != claimantAgentID {
			result = &claimTaskResult{Success: false, Reason: "already_claimed", Task: cur}
			return nil
		}
		if cur.Status == "completed" {
			result = &claimTaskResult{Success: false, Reason: "already_resolved", Task: cur}
			return nil
		}
		all, err := v2ListTasks(taskListID)
		if err != nil {
			return err
		}
		var unresolved []string
		for _, bid := range cur.BlockedBy {
			for _, t := range all {
				if t.ID == bid && t.Status != "completed" {
					unresolved = append(unresolved, bid)
					break
				}
			}
		}
		if len(unresolved) > 0 {
			result = &claimTaskResult{Success: false, Reason: "blocked", Task: cur, BlockedByTasks: unresolved}
			return nil
		}
		updated, err := v2UpdateTaskFields(taskListID, taskID, map[string]any{"owner": claimantAgentID})
		if err != nil {
			return err
		}
		result = &claimTaskResult{Success: true, Task: updated}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// claimTaskWithBusyCheck claims a task and checks if the agent is busy. Uses list-level lock.
// Mirrors TS claimTaskWithBusyCheck in utils/tasks.ts.
func claimTaskWithBusyCheck(taskListID, taskID, claimantAgentID string) (*claimTaskResult, error) {
	existing, err := v2GetTask(taskListID, taskID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &claimTaskResult{Success: false, Reason: "task_not_found"}, nil
	}
	lockPath := v2ListLockPath(taskListID)
	var result *claimTaskResult
	err = withListExclusiveLock(lockPath, func() error {
		all, err := v2ListTasks(taskListID)
		if err != nil {
			return err
		}
		var cur *v2Task
		for _, t := range all {
			if t.ID == taskID {
				cur = t
				break
			}
		}
		if cur == nil {
			result = &claimTaskResult{Success: false, Reason: "task_not_found"}
			return nil
		}
		if cur.Owner != "" && cur.Owner != claimantAgentID {
			result = &claimTaskResult{Success: false, Reason: "already_claimed", Task: cur}
			return nil
		}
		if cur.Status == "completed" {
			result = &claimTaskResult{Success: false, Reason: "already_resolved", Task: cur}
			return nil
		}
		var unresolved []string
		for _, bid := range cur.BlockedBy {
			for _, t := range all {
				if t.ID == bid && t.Status != "completed" {
					unresolved = append(unresolved, bid)
					break
				}
			}
		}
		if len(unresolved) > 0 {
			result = &claimTaskResult{Success: false, Reason: "blocked", Task: cur, BlockedByTasks: unresolved}
			return nil
		}
		var busyWith []string
		for _, t := range all {
			if t.ID != taskID && t.Status != "completed" && t.Owner == claimantAgentID {
				busyWith = append(busyWith, t.ID)
			}
		}
		if len(busyWith) > 0 {
			result = &claimTaskResult{Success: false, Reason: "agent_busy", BusyWithTasks: busyWith}
			return nil
		}
		updated, err := v2UpdateTaskFields(taskListID, taskID, map[string]any{"owner": claimantAgentID})
		if err != nil {
			return err
		}
		result = &claimTaskResult{Success: true, Task: updated}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// unassignTeammateTasks unassigns all non-completed tasks owned by a teammate and resets them to pending.
// Mirrors TS unassignTeammateTasks in utils/tasks.ts.
func unassignTeammateTasks(teamName, teammateID, teammateName, reason string) (json.RawMessage, string, error) {
	all, err := v2ListTasks(teamName)
	if err != nil {
		return nil, "", err
	}
	type unassignedItem struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	var unassigned []unassignedItem
	for _, t := range all {
		if t.Status == "completed" {
			continue
		}
		if t.Owner == teammateID || t.Owner == teammateName {
			_, err := v2UpdateTaskFields(teamName, t.ID, map[string]any{
				"owner":  nil,
				"status": "pending",
			})
			if err != nil {
				return nil, "", err
			}
			unassigned = append(unassigned, unassignedItem{ID: t.ID, Subject: t.Subject})
		}
	}
	var reasonText string
	switch reason {
	case "terminated":
		reasonText = "was terminated"
	case "shutdown":
		reasonText = "has shut down"
	default:
		reasonText = "is no longer available"
	}
	payload, _ := json.Marshal(map[string]any{"unassignedTasks": unassigned})
	if len(unassigned) > 0 {
		var ids []string
		for _, u := range unassigned {
			ids = append(ids, "#"+u.ID+" \""+u.Subject+"\"")
		}
		notification := fmt.Sprintf(
			"%s %s. %d task(s) were unassigned: %s. Use TaskList to check availability and TaskUpdate with owner to reassign them to idle teammates.",
			teammateName, reasonText, len(unassigned), strings.Join(ids, ", "),
		)
		return payload, notification, nil
	}
	return payload, "", nil
}
