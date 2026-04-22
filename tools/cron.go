package tools

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

const maxCronJobs = 50

type cronTask struct {
	ID          string `json:"id"`
	Cron        string `json:"cron"`
	Prompt      string `json:"prompt"`
	CreatedAt   int64  `json:"createdAt"`
	LastFiredAt *int64 `json:"lastFiredAt,omitempty"`
	Recurring   bool   `json:"recurring,omitempty"`
	SessionOnly bool   `json:"-"`
}

type cronFile struct {
	Tasks []cronTask `json:"tasks"`
}

var (
	cronMu     sync.Mutex
	sessionBuf []cronTask // in-memory session-only jobs (durable=false)
)

func cronFilePath(projectRoot string) string {
	pr := strings.TrimSpace(projectRoot)
	if pr == "" {
		pr = "."
	}
	return filepath.Join(pr, ".claude", "scheduled_tasks.json")
}

func parseCron(expr string) (cron.Schedule, error) {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return p.Parse(strings.TrimSpace(expr))
}

func nextCronWithinYear(expr string, from time.Time) (time.Time, bool) {
	s, err := parseCron(expr)
	if err != nil {
		return time.Time{}, false
	}
	end := from.AddDate(1, 0, 0)
	t := from
	for i := 0; i < 5000; i++ {
		next := s.Next(t)
		if next.IsZero() {
			return time.Time{}, false
		}
		if !next.Before(end) {
			return time.Time{}, false
		}
		if next.After(from) {
			return next, true
		}
		// next == from (rare); advance
		t = next.Add(time.Nanosecond)
	}
	return time.Time{}, false
}

func humanCron(expr string) string {
	next, ok := nextCronWithinYear(expr, time.Now())
	if !ok {
		return expr
	}
	return fmt.Sprintf("%s (next ~ %s local)", expr, next.Format(time.RFC3339))
}

func readCronFile(projectRoot string) ([]cronTask, error) {
	path := cronFilePath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f cronFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil
	}
	out := make([]cronTask, 0, len(f.Tasks))
	for _, t := range f.Tasks {
		if _, err := parseCron(t.Cron); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func writeCronFile(projectRoot string, tasks []cronTask) error {
	path := cronFilePath(projectRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if tasks == nil {
		tasks = []cronTask{}
	}
	body, err := json.MarshalIndent(cronFile{Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(body, '\n'), 0o644)
}

func listMerged(projectRoot string) []cronTask {
	fileTasks, _ := readCronFile(projectRoot)
	cronMu.Lock()
	sess := append([]cronTask(nil), sessionBuf...)
	cronMu.Unlock()
	out := make([]cronTask, 0, len(fileTasks)+len(sess))
	out = append(out, fileTasks...)
	for _, t := range sess {
		x := t
		out = append(out, x)
	}
	return out
}

// CronCreateFromJSON adds a cron job (durable → file, else session memory).
func CronCreateFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		Cron      string `json:"cron"`
		Prompt    string `json:"prompt"`
		Recurring *bool  `json:"recurring"`
		Durable   *bool  `json:"durable"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	expr := strings.TrimSpace(in.Cron)
	if expr == "" {
		return "", true, fmt.Errorf("cron is required")
	}
	if _, err := parseCron(expr); err != nil {
		return "", true, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	if _, ok := nextCronWithinYear(expr, time.Now()); !ok {
		return "", true, fmt.Errorf("cron %q does not match any time in the next year", expr)
	}
	all := listMerged(trimProjectRoot(c.ProjectRoot))
	if len(all) >= maxCronJobs {
		return "", true, fmt.Errorf("too many scheduled jobs (max %d)", maxCronJobs)
	}
	recurring := true
	if in.Recurring != nil {
		recurring = *in.Recurring
	}
	durable := false
	if in.Durable != nil {
		durable = *in.Durable
	}
	id := randomCronID()
	task := cronTask{
		ID:        id,
		Cron:      expr,
		Prompt:    in.Prompt,
		CreatedAt: time.Now().UnixMilli(),
		Recurring: recurring,
	}
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	effectiveDurable := durable
	if effectiveDurable {
		fileTasks, err := readCronFile(pr)
		if err != nil {
			return "", true, err
		}
		fileTasks = append(fileTasks, task)
		if err := writeCronFile(pr, fileTasks); err != nil {
			return "", true, err
		}
	} else {
		task.SessionOnly = true
		cronMu.Lock()
		sessionBuf = append(sessionBuf, task)
		cronMu.Unlock()
	}
	out := map[string]any{
		"data": map[string]any{
			"id":            id,
			"humanSchedule": humanCron(expr),
			"recurring":     recurring,
			"durable":       effectiveDurable,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func randomCronID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// CronDeleteFromJSON removes a job from session buffer and/or disk file.
func CronDeleteFromJSON(raw []byte, c Config) (string, bool, error) {
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return "", true, fmt.Errorf("id is required")
	}
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	removed := false
	cronMu.Lock()
	n := sessionBuf[:0]
	for _, t := range sessionBuf {
		if t.ID == id {
			removed = true
			continue
		}
		n = append(n, t)
	}
	sessionBuf = n
	cronMu.Unlock()
	fileTasks, err := readCronFile(pr)
	if err == nil {
		kept := make([]cronTask, 0, len(fileTasks))
		for _, t := range fileTasks {
			if t.ID == id {
				removed = true
				continue
			}
			kept = append(kept, t)
		}
		if len(kept) != len(fileTasks) {
			_ = writeCronFile(pr, kept)
		}
	}
	if !removed {
		return "", true, fmt.Errorf("no cron job with id %q", id)
	}
	out := map[string]any{"data": map[string]any{"id": id}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// CronListFromJSON lists merged file + session jobs.
func CronListFromJSON(_ []byte, c Config) (string, bool, error) {
	pr := trimProjectRoot(c.ProjectRoot)
	if pr == "" {
		pr = c.WorkDir
	}
	all := listMerged(pr)
	jobs := make([]map[string]any, 0, len(all))
	for _, t := range all {
		j := map[string]any{
			"id":            t.ID,
			"cron":          t.Cron,
			"humanSchedule": humanCron(t.Cron),
			"prompt":        t.Prompt,
		}
		if t.Recurring {
			j["recurring"] = true
		}
		if t.SessionOnly {
			j["durable"] = false
		}
		jobs = append(jobs, j)
	}
	out := map[string]any{"data": map[string]any{"jobs": jobs}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
