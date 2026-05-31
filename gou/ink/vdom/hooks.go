package vdom

import "reflect"

// UseState creates a stateful value and a setter for the given fiber.
func UseState[T any](ctx *Context, fiber *Fiber, initial T) (T, func(T)) {
	idx := ctx.hookIndex
	ctx.hookIndex++
	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}
	hs := &fiber.hooks[idx]
	if hs.effectRun {
		return hs.state.(T), func(v T) {
			hs.state = v
			if ctx.schedule != nil {
				ctx.schedule()
			}
		}
	}
	hs.state = initial
	hs.effectRun = true
	return initial, func(v T) {
		hs.state = v
		if ctx.schedule != nil {
			ctx.schedule()
		}
	}
}

// UseEffect runs a side-effect function, re-running when deps change.
func UseEffect(ctx *Context, fiber *Fiber, fn func() func(), deps []interface{}) {
	idx := ctx.hookIndex
	ctx.hookIndex++
	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}
	hs := &fiber.hooks[idx]
	if hs.effectRun && depsEqual(hs.deps, deps) {
		return
	}
	if hs.cleanup != nil {
		hs.cleanup()
	}
	hs.deps = cloneDeps(deps)
	hs.cleanup = fn()
	hs.effectRun = true
}

// UseMemo memoizes the result of a computation, recomputing when deps change.
func UseMemo(ctx *Context, fiber *Fiber, fn func() interface{}, deps []interface{}) interface{} {
	idx := ctx.hookIndex
	ctx.hookIndex++
	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}
	hs := &fiber.hooks[idx]
	if hs.effectRun && depsEqual(hs.deps, deps) {
		return hs.memoized
	}
	hs.deps = cloneDeps(deps)
	hs.memoized = fn()
	hs.effectRun = true
	return hs.memoized
}

// UseCallback memoizes a function reference, re-creating when deps change.
func UseCallback(ctx *Context, fiber *Fiber, fn interface{}, deps []interface{}) interface{} {
	idx := ctx.hookIndex
	ctx.hookIndex++
	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}
	hs := &fiber.hooks[idx]
	if hs.effectRun && depsEqual(hs.deps, deps) {
		return hs.memoized
	}
	hs.deps = cloneDeps(deps)
	hs.memoized = fn
	hs.effectRun = true
	return fn
}

func depsEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func cloneDeps(deps []interface{}) []interface{} {
	if deps == nil {
		return nil
	}
	out := make([]interface{}, len(deps))
	copy(out, deps)
	return out
}
