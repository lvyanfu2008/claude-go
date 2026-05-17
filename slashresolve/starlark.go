package slashresolve

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.starlark.net/starlark"
)

// StarlarkContext holds all data exposed to Starlark scripts via builtin functions.
type StarlarkContext struct {
	Cwd           string
	SessionMemory string
	UserMessages  []string
	SessionID     string
	SkillRoot     string
	// timeout for script execution (default 5s)
	Timeout time.Duration
}

const (
	starlarkDefaultTimeout = 5 * time.Second
	starlarkMaxFileRead    = 1 << 20 // 1MB
)

// ExecuteStarlarkSkill compiles and executes a Starlark skill script.
// The script must define a function `resolve(args)` that returns a string.
// Builtin helpers (env, cwd, read_file, etc.) are available as predeclared globals.
// pathHint is used as the filename for error messages.
func ExecuteStarlarkSkill(script string, args string, sctx *StarlarkContext, pathHint string) (string, error) {
	if sctx == nil {
		sctx = &StarlarkContext{}
	}
	timeout := sctx.Timeout
	if timeout <= 0 {
		timeout = starlarkDefaultTimeout
	}

	key := pathHint
	if key == "" {
		key = "skill.star"
	}

	predeclared := buildStarlarkPredeclared()
	thread := newStarlarkThread(sctx)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			thread.Cancel("execution timed out")
		}
	}()

	globals, err := starlark.ExecFile(thread, key, script, predeclared)
	if err != nil {
		return "", fmt.Errorf("starlark exec: %w", err)
	}

	resolveFn := globals["resolve"]
	if resolveFn == nil {
		return "", fmt.Errorf("starlark: script must define a 'resolve' function")
	}

	result, err := starlark.Call(thread, resolveFn, starlark.Tuple{starlark.String(args)}, nil)
	if err != nil {
		return "", fmt.Errorf("starlark resolve: %w", err)
	}

	s, ok := starlark.AsString(result)
	if !ok {
		return "", fmt.Errorf("starlark: resolve() must return a string, got %s", result.Type())
	}
	return s, nil
}

func newStarlarkThread(sctx *StarlarkContext) *starlark.Thread {
	t := &starlark.Thread{
		Name: "skill",
	}
	t.SetLocal("sctx", sctx)
	return t
}

// buildStarlarkPredeclared returns predeclared globals for Starlark skill scripts.
// Scripts call these as top-level functions: env("HOME"), cwd(), read_file("x.txt"), etc.
func buildStarlarkPredeclared() starlark.StringDict {
	return starlark.StringDict{
		"env":                    starlark.NewBuiltin("env", starlarkEnvBuiltin),
		"cwd":                    starlark.NewBuiltin("cwd", starlarkCwdBuiltin),
		"session_id":             starlark.NewBuiltin("session_id", starlarkSessionIDBuiltin),
		"session_memory":         starlark.NewBuiltin("session_memory", starlarkSessionMemoryBuiltin),
		"user_type":              starlark.NewBuiltin("user_type", starlarkUserTypeBuiltin),
		"is_demo":                starlark.NewBuiltin("is_demo", starlarkIsDemoBuiltin),
		"feature_enabled":        starlark.NewBuiltin("feature_enabled", starlarkFeatureEnabledBuiltin),
		"read_file":              starlark.NewBuiltin("read_file", starlarkReadFileBuiltin),
		"file_exists":            starlark.NewBuiltin("file_exists", starlarkFileExistsBuiltin),
		"list_dir":               starlark.NewBuiltin("list_dir", starlarkListDirBuiltin),
		"skill_root":             starlark.NewBuiltin("skill_root", starlarkSkillRootBuiltin),
		"session_debug_log_path": starlark.NewBuiltin("session_debug_log_path", starlarkSessionDebugLogPathBuiltin),
		"str_contains":           starlark.NewBuiltin("str_contains", starlarkStrContainsBuiltin),
		"str_has_prefix":         starlark.NewBuiltin("str_has_prefix", starlarkStrHasPrefixBuiltin),
		"str_has_suffix":         starlark.NewBuiltin("str_has_suffix", starlarkStrHasSuffixBuiltin),
		"str_join":               starlark.NewBuiltin("str_join", starlarkStrJoinBuiltin),
	}
}

func getSctx(thread *starlark.Thread) *StarlarkContext {
	v := thread.Local("sctx")
	if v == nil {
		return &StarlarkContext{}
	}
	return v.(*StarlarkContext)
}

func fileModTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
