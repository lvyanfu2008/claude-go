package store

import "testing"

func TestSelectorBasic(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		return a.Get() + b.Get()
	})
	if sel.Get() != 30 {
		t.Fatalf("expected 30, got %v", sel.Get())
	}
}

func TestSelectorMemo(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	computes := 0
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		computes++
		return a.Get() + b.Get()
	})
	_ = sel.Get()
	_ = sel.Get()
	_ = sel.Get()
	if computes != 1 {
		t.Fatalf("expected 1 computation, got %d", computes)
	}
}

func TestSelectorRecomputeOnChange(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	computes := 0
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		computes++
		return a.Get() + b.Get()
	})
	_ = sel.Get()
	a.Set(100)
	_ = sel.Get()
	if computes != 2 {
		t.Fatalf("expected 2 computations, got %d", computes)
	}
}

func TestSelectorNoRecomputeOnSameValue(t *testing.T) {
	a := NewAtom(10)
	computes := 0
	sel := NewSelector([]AtomReader{a}, func() interface{} {
		computes++
		return a.Get() * 2
	})
	_ = sel.Get()
	a.Set(10)
	_ = sel.Get()
	if computes != 2 {
		t.Fatalf("expected 2 computations (version-based invalidation), got %d", computes)
	}
}

func TestSelectorTyped(t *testing.T) {
	a := NewAtom("hello")
	b := NewAtom("world")
	sel := NewTypedSelector([]AtomReader{a, b}, func() string {
		return a.Get() + " " + b.Get()
	})
	if sel.Get() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", sel.Get())
	}
}
