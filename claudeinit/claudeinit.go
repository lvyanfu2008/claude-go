// Package claudeinit is a source-level port-in-progress of [src/entrypoints/init.ts] init().
// See [docs/plans/go-init-port.md] for the parity matrix (required / stub / explicit_gap).
package claudeinit

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"goc/ccb-engine/settingsfile"
)

// Options configures Init behavior (mirrors non-interactive / cwd aspects of TS CLI).
type Options struct {
	// NonInteractive when true matches TS getIsNonInteractiveSession: config errors return instead of UI.
	NonInteractive bool
	// WorkingDir if non-empty is chdir'd before init (best-effort).
	WorkingDir string
}

var initOnce sync.Once

// Init runs the Go-side init sequence once per process (TS memoize(init) equivalent).
func Init(ctx context.Context, opts Options) error {
	var err error
	initOnce.Do(func() {
		err = initImpl(ctx, opts)
	})
	return err
}

func initImpl(ctx context.Context, opts Options) error {
	_ = ctx
	if wd := opts.WorkingDir; wd != "" {
		if e := os.Chdir(wd); e != nil {
			return fmt.Errorf("claudeinit: chdir %q: %w", wd, e)
		}
	}

	if err := phase1ConfigAndEnv(opts); err != nil {
		return err
	}
	registerGracefulShutdown() // matrix: no global signals (TUI owns SIGINT)
	phase2AsyncSideEffects()
	if err := phase3Network(); err != nil {
		return err
	}
	if err := phase4FSAndPlatform(); err != nil {
		return err
	}
	recordInitWallTimeMS()
	return nil
}

// ResetForTesting resets the process-wide once (tests only).
func ResetForTesting() {
	initOnce = sync.Once{}
	resetRepoDetectForTesting()
	preconnectResetForTesting()
}

// ProjectRoot returns the project root last resolved by settings merge (empty before successful init path).
func ProjectRoot() string {
	return settingsfile.ProjectRootLastResolved()
}

// InitWallTimeMS returns wall ms recorded at end of Init (-1 if unset).
func InitWallTimeMS() int64 {
	s := os.Getenv("CLAUDE_CODE_GO_INIT_WALL_MS")
	if s == "" {
		return -1
	}
	var ms int64
	_, _ = fmt.Sscanf(s, "%d", &ms)
	return ms
}

func recordInitWallTimeMS() {
	_ = os.Setenv("CLAUDE_CODE_GO_INIT_WALL_MS", fmt.Sprintf("%d", time.Now().UnixMilli()))
}
