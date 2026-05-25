package tools

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

const (
	cronTickInterval     = 1 * time.Second
	cronLockProbeInterval = 5 * time.Second
	cronRecurringMaxAge   = 7 * 24 * time.Hour
	cronMaxJitterMs       = 60_000 // 1 minute max jitter
)

// CronScheduler runs a background loop that fires due cron jobs.
// Only one scheduler per project root is active (file-lock coordination).
type CronScheduler struct {
	mu          sync.Mutex
	projectRoot string
	lockPath    string
	lock        *flock.Flock
	ticker      *time.Ticker
	stopCh      chan struct{}
	running     bool
	onFire      func(ctx context.Context, prompt string)
	cancel      context.CancelFunc
}

// NewCronScheduler creates a scheduler. projectRoot determines the lock file
// and scheduled_tasks.json location. onFire is called when a job fires.
func NewCronScheduler(projectRoot string, onFire func(ctx context.Context, prompt string)) *CronScheduler {
	pr := strings.TrimSpace(projectRoot)
	if pr == "" {
		pr = "."
	}
	abs, err := filepath.Abs(pr)
	if err != nil {
		abs = pr
	}
	lockPath := filepath.Join(abs, ".harness", "scheduled_tasks.lock")
	return &CronScheduler{
		projectRoot: abs,
		lockPath:    lockPath,
		lock:        flock.New(lockPath),
		stopCh:      make(chan struct{}),
		onFire:      onFire,
	}
}

// Start begins the scheduler loop in a background goroutine.
// Returns immediately. Safe to call multiple times (no-op if already running).
func (cs *CronScheduler) Start(ctx context.Context) {
	cs.mu.Lock()
	if cs.running {
		cs.mu.Unlock()
		return
	}
	cs.running = true
	cs.mu.Unlock()

	var sctx context.Context
	sctx, cs.cancel = context.WithCancel(ctx)
	go cs.run(sctx)
}

// Stop gracefully shuts down the scheduler.
func (cs *CronScheduler) Stop() {
	cs.mu.Lock()
	if !cs.running {
		cs.mu.Unlock()
		return
	}
	cs.running = false
	cs.mu.Unlock()
	if cs.cancel != nil {
		cs.cancel()
	}
	close(cs.stopCh)
	if cs.ticker != nil {
		cs.ticker.Stop()
	}
	cs.releaseLock()
}

func (cs *CronScheduler) releaseLock() {
	if cs.lock != nil && cs.lock.Locked() {
		_ = cs.lock.Unlock()
	}
}

func (cs *CronScheduler) run(ctx context.Context) {
	// Try to acquire the lock. If we can't, probe periodically.
	for {
		locked, err := cs.lock.TryLock()
		if err == nil && locked {
			defer cs.releaseLock()
			cs.detectMissedTasks(ctx)
			cs.scheduleLoop(ctx)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-cs.stopCh:
			return
		case <-time.After(cronLockProbeInterval):
		}
	}
}

func (cs *CronScheduler) scheduleLoop(ctx context.Context) {
	cs.ticker = time.NewTicker(cronTickInterval)
	defer cs.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopCh:
			return
		case <-cs.ticker.C:
			cs.tick(ctx)
		}
	}
}

func (cs *CronScheduler) tick(ctx context.Context) {
	all := listMerged(cs.projectRoot)
	now := time.Now()
	for _, task := range all {
		cs.checkAndFire(ctx, task, now)
	}
}

func (cs *CronScheduler) checkAndFire(ctx context.Context, task cronTask, now time.Time) {
	s, err := parseCron(task.Cron)
	if err != nil {
		return
	}
	next := s.Next(now)
	if next.IsZero() {
		return
	}
	// Skip if the next fire time is after now (not due yet).
	if next.After(now) {
		return
	}

	jitter := cs.jitterForTask(task)
	fireTime := next.Add(jitter)
	if now.Before(fireTime) {
		return
	}

	// Fire the task.
	cs.fireTask(ctx, task)

	// Update bookkeeping.
	nowMs := now.UnixMilli()
	task.LastFiredAt = &nowMs
	cs.persistFire(task)

	// One-shot: delete after firing.
	if !task.Recurring {
		cs.deleteTask(task.ID)
		return
	}

	// Recurring: delete if expired.
	if time.Since(time.UnixMilli(task.CreatedAt)) > cronRecurringMaxAge {
		cs.deleteTask(task.ID)
	}
}

func (cs *CronScheduler) fireTask(ctx context.Context, task cronTask) {
	if cs.onFire != nil {
		cs.onFire(ctx, task.Prompt)
	}
}

func (cs *CronScheduler) persistFire(task cronTask) {
	fileTasks, err := readCronFile(cs.projectRoot)
	if err != nil {
		return
	}
	nowMs := time.Now().UnixMilli()
	found := false
	for i := range fileTasks {
		if fileTasks[i].ID == task.ID {
			fileTasks[i].LastFiredAt = &nowMs
			found = true
			break
		}
	}
	if found {
		_ = writeCronFile(cs.projectRoot, fileTasks)
	}
}

func (cs *CronScheduler) deleteTask(id string) {
	cronMu.Lock()
	n := sessionBuf[:0]
	for _, t := range sessionBuf {
		if t.ID == id {
			continue
		}
		n = append(n, t)
	}
	sessionBuf = n
	cronMu.Unlock()

	fileTasks, err := readCronFile(cs.projectRoot)
	if err != nil {
		return
	}
	kept := make([]cronTask, 0, len(fileTasks))
	for _, t := range fileTasks {
		if t.ID == id {
			continue
		}
		kept = append(kept, t)
	}
	if len(kept) != len(fileTasks) {
		_ = writeCronFile(cs.projectRoot, kept)
	}
}

func (cs *CronScheduler) detectMissedTasks(ctx context.Context) {
	all := listMerged(cs.projectRoot)
	now := time.Now()
	for _, task := range all {
		if task.LastFiredAt != nil {
			continue // Already fired at least once, skip.
		}
		s, err := parseCron(task.Cron)
		if err != nil {
			continue
		}
		// Compute the most recent fire time before now.
		// Walk back from 'now' to find the last scheduled time.
		t := now.Add(-cronRecurringMaxAge)
		if created := time.UnixMilli(task.CreatedAt); created.After(t) {
			t = created
		}
		missed := false
		for i := 0; i < 5000; i++ {
			next := s.Next(t)
			if next.IsZero() || next.After(now) {
				break
			}
			if next.After(t) {
				// This time is between task creation/last check and now.
				// Only fire once for the most recent missed slot.
				missed = true
			}
			t = next.Add(time.Second)
		}
		if missed {
			// Fire once for missed one-shot tasks; skip recurring to avoid flood.
			if !task.Recurring {
				cs.fireTask(ctx, task)
				cs.deleteTask(task.ID)
			}
		}
	}
}

// jitterForTask returns a deterministic jitter duration based on the task ID hash.
// Range: [0, min(60s, 10% of cron interval)].
func (cs *CronScheduler) jitterForTask(task cronTask) time.Duration {
	s, err := parseCron(task.Cron)
	if err != nil {
		return 0
	}
	// Estimate cron interval: time between two consecutive Next() calls.
	now := time.Now()
	next1 := s.Next(now)
	if next1.IsZero() {
		return 0
	}
	next2 := s.Next(next1.Add(time.Second))
	if next2.IsZero() {
		return 0
	}
	interval := next2.Sub(next1)

	// Jitter cap: 10% of interval, clamped to [1s, 60s].
	maxJitter := interval / 10
	if maxJitter < time.Second {
		maxJitter = time.Second
	}
	if maxJitter > cronMaxJitterMs*time.Millisecond {
		maxJitter = cronMaxJitterMs * time.Millisecond
	}

	// Deterministic hash of task ID.
	h := sha256.Sum256([]byte(task.ID))
	hashMs := binary.BigEndian.Uint64(h[:8]) % uint64(maxJitter.Milliseconds())
	return time.Duration(hashMs) * time.Millisecond
}

// ensureLockDir creates the parent directory for the lock file.
func ensureLockDir(lockPath string) error {
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return nil
}
