package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// WorkflowEngine executes Workflow scripts using a Goja JS runtime.
// It implements an async event loop: the VM runs on a single goroutine,
// agent() calls spawn goroutines that send results back via a channel.
type WorkflowEngine struct {
	runID  string
	taskID string
}

// NewEngine creates a new WorkflowEngine.
func NewEngine() *WorkflowEngine {
	return &WorkflowEngine{
		runID: GenerateRunID(),
	}
}

// RunID returns the workflow's run ID (for resume support).
func (e *WorkflowEngine) RunID() string { return e.runID }

// TaskID returns the task ID for the underlying v2 task.
func (e *WorkflowEngine) TaskID() string { return e.taskID }

// Execute runs a workflow script with the given config.
// Returns the workflow's final result as a string.
func (e *WorkflowEngine) Execute(ctx context.Context, script string, cfg EngineConfig) (string, error) {
	// 1. Parse meta
	meta, err := ParseMeta(script)
	if err != nil {
		return "", fmt.Errorf("workflow: parse: %w", err)
	}

	// 2. Create task in v2 system
	e.taskID = ""
	if taskCreateFn != nil {
		tid, cerr := taskCreateFn(TaskListID(cfg), meta.Name, meta.Description)
		if cerr == nil {
			e.taskID = tid
			_ = updateTaskInSystem(TaskListID(cfg), e.taskID, "in_progress")
		}
	}

	// 3. Create run state with args from engine config
	state := NewRunState(e.runID, *meta, cfg.Args, cfg.ProgressCallback, cfg.WorkflowProgressCallback)

	// 3.5 Set up per-workflow debug log dir
	if cfg.LogDir != "" {
		dir := filepath.Join(cfg.LogDir, e.runID)
		os.MkdirAll(dir, 0o755)
		SetWorkflowLogDir(dir)
	}

	// 4. Set up Goja runtime
	completionCh := make(chan completion, 100)
	vm, err := newRuntime(state, cfg, completionCh)
	if err != nil {
		e.fail(cfg, state, err)
		return "", fmt.Errorf("workflow: runtime: %w", err)
	}

	// 5. Pre-process script: strip exports
	body := StripExports(script)

	// 6. Wrap in async IIFE
	wrapped := "(async () => {\n" + body + "\n})()"

	// 7. Execute script in VM; retry with ${} escaping on SyntaxError
	val, err := vm.RunString(wrapped)
	if err != nil && strings.Contains(err.Error(), "SyntaxError") && strings.Contains(body, "${") {
		// Only escape ${ outside of template literal backticks (preserve intentional ${} in \`...\`)
		escaped := escapeDollarBraceOutsideTemplate(body)
		if escaped != body {
			wrapped2 := "(async () => {\n" + escaped + "\n})()"
			val2, err2 := vm.RunString(wrapped2)
			if err2 == nil {
				val = val2
				err = nil
			} else {
				errMsg := err2.Error()
				if strings.Contains(errMsg, "SyntaxError") {
					errMsg += "\nHint: your script contains '${}' which conflicts with JS template literals. " +
						"Pass file content via args instead of embedding it in the script."
				}
				e.fail(cfg, state, err2)
				return "", fmt.Errorf("workflow: script: %s", errMsg)
			}
		} else {
			e.fail(cfg, state, err)
			return "", fmt.Errorf("workflow: script: %w", err)
		}
	}
	if err != nil {
		e.fail(cfg, state, err)
		return "", fmt.Errorf("workflow: script: %w", err)
	}

	// 8. Extract the top-level promise
	promise, ok := val.Export().(*goja.Promise)
	if !ok {
		// Synchronous script — result is the value itself
		result := val.String()
		e.complete(cfg, state, result)
		return result, nil
	}

	// 9. Event loop: process until the top-level promise resolves or the context is cancelled
	for {
		// Check for abort / timeout
		select {
		case <-ctx.Done():
			state.Abort()
			e.fail(cfg, state, ctx.Err())
			return "", ctx.Err()
		case <-state.Aborted():
			// Abort was triggered internally
			e.fail(cfg, state, fmt.Errorf("workflow aborted"))
			return "", fmt.Errorf("workflow aborted")
		default:
		}

		// Check top-level promise state
		switch promise.State() {
		case goja.PromiseStateFulfilled:
			result := stringifyResult(vm, promise.Result())
			e.complete(cfg, state, result)
			return result, nil

		case goja.PromiseStateRejected:
			errVal := promise.Result()
			errStr := "unknown error"
			if errVal != nil {
				errStr = errVal.String()
			}
			err := fmt.Errorf("workflow: %s", errStr)
			e.fail(cfg, state, err)
			return "", err
		}

		// Process pending completions from agent goroutines
		select {
		case comp := <-completionCh:
			if comp.err != nil {
				comp.reject(vm.ToValue(comp.err.Error()))
			} else {
				// Convert raw Go value to goja.Value on VM goroutine (Goja not goroutine-safe)
			comp.resolve(vm.ToValue(comp.rawResult))
			}
			// Trigger promise microtask processing
			vm.RunString("void 0")

		case <-ctx.Done():
			state.Abort()
			e.fail(cfg, state, ctx.Err())
			return "", ctx.Err()

		case <-state.Aborted():
			e.fail(cfg, state, fmt.Errorf("workflow aborted"))
			return "", fmt.Errorf("workflow aborted")

		default:
			// No completions pending - check one more time
			// and if still not done, sleep briefly
			switch promise.State() {
			case goja.PromiseStateFulfilled:
				result := stringifyResult(vm, promise.Result())
				e.complete(cfg, state, result)
				return result, nil
			case goja.PromiseStateRejected:
				errVal := promise.Result()
				errStr := "unknown error"
				if errVal != nil {
					errStr = errVal.String()
				}
				err := fmt.Errorf("workflow: %s", errStr)
				e.fail(cfg, state, err)
				return "", err
			}

			// Brief sleep to avoid busy-waiting, with context check
			select {
			case <-time.After(10 * time.Millisecond):
			case <-ctx.Done():
				state.Abort()
				return "", ctx.Err()
			case <-state.Aborted():
				return "", fmt.Errorf("workflow aborted")
			case comp := <-completionCh:
				if comp.err != nil {
					comp.reject(vm.ToValue(comp.err.Error()))
				} else {
					comp.resolve(vm.ToValue(comp.rawResult))
				}
				vm.RunString("void 0")
			}
		}
	}
}

// complete marks the workflow as completed.
func (e *WorkflowEngine) complete(cfg EngineConfig, state *RunState, result string) {
	if e.taskID != "" {
		CompleteTask(TaskListID(cfg), e.taskID, state.Meta.Name, result, cfg.NotificationCallback)
	}
}

// fail marks the workflow as failed.
func (e *WorkflowEngine) fail(cfg EngineConfig, state *RunState, err error) {
	if e.taskID != "" {
		FailTask(TaskListID(cfg), e.taskID, state.Meta.Name, err, cfg.NotificationCallback)
	}
}

// TaskListID computes a stable task list identifier from the engine config.
func TaskListID(cfg EngineConfig) string {
	if cfg.SessionID != "" {
		return cfg.SessionID
	}
	return "default-session"
}

// escapeDollarBraceOutsideTemplate escapes ${ → \${ only outside template literal backticks.
// Preserves intentional ${} inside \`...\` (e.g. \`Hello ${name}\` stays intact).
func escapeDollarBraceOutsideTemplate(s string) string {
	var out strings.Builder
	inTemplate := false
	inString := false
	var stringChar byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' && i+1 < len(s) {
			out.WriteByte(ch)
			i++
			out.WriteByte(s[i])
			continue
		}
		if ch == '`' && !inString {
			inTemplate = !inTemplate
			out.WriteByte(ch)
			continue
		}
		if (ch == '"' || ch == '\'') && !inTemplate {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				inString = false
			}
		}
		if ch == '$' && i+1 < len(s) && s[i+1] == '{' && !inTemplate {
			out.WriteString("\\${")
			i++ // skip '{'
			continue
		}
		out.WriteByte(ch)
	}
	return out.String()
}

// stringifyResult converts a Goja value to a string, using JSON.stringify for objects.
func stringifyResult(vm *goja.Runtime, val goja.Value) string {
	// For objects/arrays, use JSON.stringify; for primitives, use String()
	exported := val.Export()
	switch exported.(type) {
	case string, float64, int64, bool, nil:
		return val.String()
	default:
		// Object or array — use JSON.stringify
		if jsonFn, ok := goja.AssertFunction(vm.Get("JSON").(*goja.Object).Get("stringify")); ok {
			if result, err := jsonFn(goja.Undefined(), val); err == nil {
				s := result.String()
				if s != "undefined" {
					return s
				}
			}
		}
		return val.String()
	}
}

// GenerateRunID generates a random workflow run ID in the format wf_<hex>.
func GenerateRunID() string {
	id := randomHex(8)
	return "wf_" + id
}
