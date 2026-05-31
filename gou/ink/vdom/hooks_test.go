package vdom

import (
	"sync"
	"testing"
)

func TestUseState(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	val, setVal := UseState(ctx, f, 0)
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
	setVal(42)
	ctx.hookIndex = 0
	val2, _ := UseState(ctx, f, 0)
	if val2 != 42 {
		t.Fatalf("expected 42 after set, got %d", val2)
	}
}

func TestUseStatePreservesIndex(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	UseState(ctx, f, "first")
	UseState(ctx, f, 999)
	ctx.hookIndex = 0
	v1, _ := UseState(ctx, f, "first")
	v2, _ := UseState(ctx, f, 999)
	if v1 != "first" {
		t.Fatalf("expected first, got %v", v1)
	}
	if v2 != 999 {
		t.Fatalf("expected 999, got %d", v2)
	}
}

func TestUseEffect(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	var mu sync.Mutex
	effects := 0
	UseEffect(ctx, f, func() func() {
		mu.Lock()
		effects++
		mu.Unlock()
		return nil
	}, nil)
	if effects != 1 {
		t.Fatalf("expected 1 effect run, got %d", effects)
	}
}

func TestUseEffectCleanup(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	cleaned := false
	UseEffect(ctx, f, func() func() {
		return func() { cleaned = true }
	}, []interface{}{1})
	ctx.hookIndex = 0
	UseEffect(ctx, f, func() func() { return func() {} }, []interface{}{2})
	if !cleaned {
		t.Error("expected cleanup to run when deps change")
	}
}

func TestUseMemo(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	computes := 0
	result := UseMemo(ctx, f, func() interface{} {
		computes++
		return 42
	}, []interface{}{1, 2})
	if result != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
	if computes != 1 {
		t.Fatalf("expected 1 compute, got %d", computes)
	}
	ctx.hookIndex = 0
	_ = UseMemo(ctx, f, func() interface{} { computes++; return 99 }, []interface{}{1, 2})
	if computes != 1 {
		t.Fatalf("expected still 1 compute (memo hit), got %d", computes)
	}
}

func TestUseMemoDepsChange(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	computes := 0
	_ = UseMemo(ctx, f, func() interface{} { computes++; return 10 }, []interface{}{1})
	ctx.hookIndex = 0
	_ = UseMemo(ctx, f, func() interface{} { computes++; return 20 }, []interface{}{2})
	if computes != 2 {
		t.Fatalf("expected 2 computes (deps changed), got %d", computes)
	}
}

func TestUseCallback(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	fn := func() int { return 1 }
	result := UseCallback(ctx, f, fn, []interface{}{1, 2})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if r, ok := result.(func() int); !ok || r() != 1 {
		t.Fatal("expected function that returns 1")
	}
	ctx.hookIndex = 0
	result2 := UseCallback(ctx, f, fn, []interface{}{1, 2})
	if result2 == nil {
		t.Fatal("expected non-nil memo result")
	}
	if r, ok := result2.(func() int); !ok || r() != 1 {
		t.Fatal("expected function that returns 1 on memo hit")
	}
}

func TestUseCallbackDepsChange(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}
	fn1 := func() int { return 1 }
	fn2 := func() int { return 2 }
	UseCallback(ctx, f, fn1, []interface{}{1})
	ctx.hookIndex = 0
	result2 := UseCallback(ctx, f, fn2, []interface{}{2})
	if result2 == nil {
		t.Fatal("expected non-nil result")
	}
	if r, ok := result2.(func() int); !ok || r() != 2 {
		t.Fatal("expected new function returning 2 when deps change")
	}
}
