package memoize

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoizeWithTTLAsync_coldCoalesce(t *testing.T) {
	var calls int32
	f := func(n int) (int, error) {
		atomic.AddInt32(&calls, 1)
		return n * 3, nil
	}
	key := func(n int) string { return "same" }
	m, _ := MemoizeWithTTLAsync(f, key, time.Hour)
	var x, y int
	var e1, e2 error
	var w sync.WaitGroup
	w.Add(2)
	go func() { x, e1 = m(1); w.Done() }()
	go func() { y, e2 = m(1); w.Done() }()
	w.Wait()
	if e1 != nil || e2 != nil {
		t.Fatalf("err %v %v", e1, e2)
	}
	if x != 3 || y != 3 {
		t.Fatalf("want 3, got %d %d", x, y)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected one f call, got %d", calls)
	}
}

func TestMemoizeWithTTLAsync_errorNotCached(t *testing.T) {
	var calls int32
	f := func(int) (int, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return 0, errors.New("x")
		}
		return 42, nil
	}
	m, _ := MemoizeWithTTLAsync(f, func(int) string { return "k" }, time.Hour)
	_, e := m(0)
	if e == nil {
		t.Fatal("expected err")
	}
	v, e2 := m(0)
	if e2 != nil || v != 42 {
		t.Fatalf("second %v %d", e2, v)
	}
}
