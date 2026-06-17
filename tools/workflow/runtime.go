package workflow

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dop251/goja"
)

// newRuntime creates a sandboxed Goja VM with all workflow globals bound.
// The VM runs on a single goroutine (the event loop goroutine). Agent completions
// are delivered via completionCh to avoid concurrent VM access.
func newRuntime(state *RunState, cfg EngineConfig, completionCh chan<- completion) (*goja.Runtime, error) {
	vm := goja.New()

	// ES6+ features
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	if err := enablePromiseSupport(vm); err != nil {
		return nil, fmt.Errorf("workflow: promise support: %w", err)
	}

	// Sandboxing
	sandboxRuntime(vm)

	// Build budget JS object
	budgetObj := buildBudgetObject(vm, state.Budget)

	// Parse args
	var argsVal goja.Value = goja.Undefined()
	if len(state.Args) > 0 {
		var parsed any
		if err := json.Unmarshal(state.Args, &parsed); err == nil {
			argsVal = vm.ToValue(parsed)
		} else {
			// Try as string
			var argsStr string
			if err := json.Unmarshal(state.Args, &argsStr); err == nil {
				argsVal = vm.ToValue(argsStr)
			}
		}
	}

	// Bind globals
	vm.Set("args", argsVal)
	vm.Set("budget", budgetObj)
	vm.Set("phase", bindPhase(state, cfg.WorkflowProgressCallback))
	vm.Set("log", bindLog(state, cfg.WorkflowProgressCallback))
	vm.Set("agent", bindAgent(vm, state, cfg, completionCh))

	// parallel and pipeline are JS-level wrappers injected as globals
	if _, err := vm.RunString(parallelJS); err != nil {
		return nil, fmt.Errorf("workflow: parallel binding: %w", err)
	}
	if _, err := vm.RunString(pipelineJS); err != nil {
		return nil, fmt.Errorf("workflow: pipeline binding: %w", err)
	}
	if _, err := vm.RunString(workflowJS); err != nil {
		return nil, fmt.Errorf("workflow: workflow binding: %w", err)
	}

	return vm, nil
}

// sandboxRuntime blocks dangerous globals in the Goja VM.
// WARNING: Do NOT override Goja built-in constructors (Date, etc.) — replacing them
// with plain functions breaks Goja's internal assumptions and causes nil pointer panics.
// Instead, only block non-constructor globals and leave the rest to the prompt documentation.
func sandboxRuntime(vm *goja.Runtime) {
	// Block dangerous browser / Node.js globals
	vm.Set("require", goja.Undefined())
	vm.Set("module", goja.Undefined())
	vm.Set("exports", goja.Undefined())
	vm.Set("process", goja.Undefined())
	vm.Set("setTimeout", goja.Undefined())
	vm.Set("setInterval", goja.Undefined())
	vm.Set("clearTimeout", goja.Undefined())
	vm.Set("clearInterval", goja.Undefined())
	vm.Set("console", goja.Undefined())
}

// enablePromiseSupport ensures Goja has Promise support enabled.
func enablePromiseSupport(vm *goja.Runtime) error {
	// Goja supports Promises natively — verify by creating one
	_, _, _ = vm.NewPromise()
	return nil
}

// buildBudgetObject creates the JS budget global: { total, spent(), remaining() }.
func buildBudgetObject(vm *goja.Runtime, b *BudgetTracker) *goja.Object {
	obj := vm.NewObject()
	if b.Total() > 0 {
		obj.Set("total", float64(b.Total()))
	} else {
		obj.Set("total", nil)
	}
	obj.Set("spent", func() float64 { return float64(b.Spent()) })
	obj.Set("remaining", func() float64 {
		r := b.Remaining()
		if r == math.MaxInt64 {
			return math.Inf(1)
		}
		return float64(r)
	})
	return obj
}

// parallelJS is the JS implementation of parallel(), injected into the Goja VM.
// Thunks are called synchronously (each thunk calls agent() which returns a Promise
// immediately). Then Promise.all wrappers catch errors and map them to null.
const parallelJS = `
var parallel = (thunks) => {
	if (!thunks || thunks.length === 0) return [];
	const promises = [];
	for (let i = 0; i < thunks.length; i++) {
		try {
			promises.push(thunks[i]());
		} catch(e) {
			promises.push(null);
		}
	}
	return Promise.all(promises.map(async (p) => {
		try { return await p; } catch(e) { return null; }
	}));
};
`

// pipelineJS is the JS implementation of pipeline(), injected into the Goja VM.
// Each item flows through all stages independently (no barrier between stages).
// Each stage callback receives (prevResult, originalItem, index).
const pipelineJS = `
var pipeline = (items, ...stages) => {
	if (!stages || stages.length === 0) return items;
	return Promise.all(items.map((item, i) => (async () => {
		let result = item;
		for (const stage of stages) {
			try {
				result = await stage(result, item, i);
			} catch(e) {
				return null;
			}
		}
		return result;
	})()));
};
`

// workflowJS is a stub for nested workflow(). Full implementation would launch
// a child workflow engine. For now it returns a helpful error.
const workflowJS = `
var workflow = (nameOrRef, args) => {
	throw new Error('Nested workflow() is not yet implemented. Use agent() for sub-tasks.');
};
`
