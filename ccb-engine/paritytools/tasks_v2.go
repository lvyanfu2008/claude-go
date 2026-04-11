package paritytools

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
)

// TS parity reference: claude-code/src/utils/tasks.ts (layout, locks, high water mark, CRUD).
// TODO(ts parity): TaskCreate/TaskUpdate hooks, teammate mailbox, verification nudge,
// GrowthBook / agent-swarm owner auto-set, completed-task blocking hooks.

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

// taskListID resolves the task list directory name (getTaskListId in TS).
// TODO(ts parity): in-process teammate / leader team name not wired in Go.
func taskListID(cfg Config) string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TASK_LIST_ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_NAME")); v != "" {
		return v
	}
	if v := strings.TrimSpace(cfg.SessionID); v != "" {
		return v
	}
	return "default"
}

func v2TasksDir(taskListID string) string {
	base := commands.ClaudeConfigHome()
	id := sanitizePathComponentV2(taskListID)
	return filepath.Join(base, "tasks", id)
}

func v2TaskPath(taskListID, taskID string) string {
	return filepath.Join(v2TasksDir(taskListID), sanitizePathComponentV2(taskID)+".json")
}

func v2ListLockPath(taskListID string) string {
	return filepath.Join(v2TasksDir(taskListID), ".lock")
}

func v2HighWaterMarkPath(taskListID string) string {
	return filepath.Join(v2TasksDir(taskListID), v2HighWaterMarkFile)
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
	dir := v2TasksDir(taskListID)
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
	switch t.Status {
	case "pending", "in_progress", "completed":
	default:
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

func v2CreateTask(taskListID string, subject, description, activeForm string, metadata map[string]any) (string, error) {
	lockPath := v2ListLockPath(taskListID)
	var newID string
	err := withListExclusiveLock(lockPath, func() error {
		highest := findHighestTaskIDV2(taskListID)
		newID = strconv.Itoa(highest + 1)
		md := metadata
		if len(md) == 0 {
			md = nil
		}
		t := &v2Task{
			ID:          newID,
			Subject:     subject,
			Description: description,
			ActiveForm:  activeForm,
			Status:      "pending",
			Blocks:      []string{},
			BlockedBy:   []string{},
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
	dir := v2TasksDir(taskListID)
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

// TaskCreateFromJSON implements TaskCreate (TS TaskCreateTool); skips executeTaskCreatedHooks.
func TaskCreateFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskCreate")
	}
	var in struct {
		Subject     string         `json:"subject"`
		Description string         `json:"description"`
		ActiveForm  string         `json:"activeForm"`
		Metadata    map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Subject) == "" {
		return "", true, fmt.Errorf("subject is required")
	}
	tid := taskListID(cfg)
	id, err := v2CreateTask(tid, in.Subject, in.Description, in.ActiveForm, in.Metadata)
	if err != nil {
		return "", true, err
	}
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
	t, err := v2GetTask(taskListID(cfg), in.TaskID)
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
	all, err := v2ListTasks(taskListID(cfg))
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

// TaskUpdateFromJSON implements TaskUpdate (TS TaskUpdateTool) without hooks, mailbox, or verification nudge.
func TaskUpdateFromJSON(ctx context.Context, raw []byte, cfg Config) (string, bool, error) {
	_ = ctx
	if !commands.TodoV2Enabled() {
		return "", true, errTodoV2Disabled("TaskUpdate")
	}
	var in struct {
		TaskID       string            `json:"taskId"`
		Subject      *string           `json:"subject"`
		Description  *string           `json:"description"`
		ActiveForm   *string           `json:"activeForm"`
		Status       *string           `json:"status"`
		Owner        *string           `json:"owner"`
		AddBlocks    []string          `json:"addBlocks"`
		AddBlockedBy []string          `json:"addBlockedBy"`
		Metadata     map[string]any    `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.TaskID) == "" {
		return "", true, fmt.Errorf("taskId is required")
	}
	tid := taskListID(cfg)
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
		case "pending", "in_progress", "completed":
		default:
			return "", true, fmt.Errorf("invalid status %q", st)
		}
		if st != existing.Status {
			// TODO(ts parity): executeTaskCompletedHooks when status becomes completed.
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

	out := map[string]any{
		"data": map[string]any{
			"success":       true,
			"taskId":        in.TaskID,
			"updatedFields": updatedFields,
		},
	}
	if statusChange != nil {
		out["data"].(map[string]any)["statusChange"] = statusChange
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
