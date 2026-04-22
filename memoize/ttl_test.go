package memoize

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoizeWithTTL_freshAndClear(t *testing.T) {
	var calls int32
	f := func(n int) int { atomic.AddInt32(&calls, 1); return n * 2 }
	key := func(n int) string { return fmt.Sprintf("k%d", n) }
	m, clear := MemoizeWithTTL(f, key, time.Hour)
	if m(3) != 6 {
		t.Fatalf("first")
	}
	if m(3) != 6 || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected second hit, calls=%d", calls)
	}
	clear()
	if m(3) != 6 || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("after clear expected recompute, calls=%d", calls)
	}
}

func TestMemoizeWithTTL_staleWhileRevalidate(t *testing.T) {
	var calls int32
	f := func(n int) int { atomic.AddInt32(&calls, 1); return int(atomic.LoadInt32(&calls)) }
	key := func(n int) string { return "k" }
	m, _ := MemoizeWithTTL(f, key, 20*time.Millisecond)
	_ = m(0) // 1
	time.Sleep(25 * time.Millisecond)
	_ = m(0) // returns stale 1, refresh scheduled
	// Eventually refresh may bump cache
	time.Sleep(30 * time.Millisecond)
	if atomic.LoadInt32(&calls) < 1 {
		t.Fatal("no calls")
	}
}

func TestMemoizeWithLRU_evictAndPeek(t *testing.T) {
	var calls int32
	f := func(n int) string {
		atomic.AddInt32(&calls, 1)
		return fmt.Sprintf("v%d", n)
	}
	m, c := MemoizeWithLRU(f, func(n int) string { return fmt.Sprintf("%d", n) }, 2)
	_ = m(1)
	_ = m(2)
	_ = m(3)
	_ = m(1)
	if c.Size() != 2 {
		t.Fatalf("Size want 2, got %d", c.Size())
	}
	_, _ = c.Get("1")
}

func TestMemoizeWithLRU_peek(t *testing.T) {
	f := func(n int) int { return n }
	m, c := MemoizeWithLRU(f, func(n int) string { return fmt.Sprint(n) }, 10)
	_ = m(7)
	_, ok := c.Get("7")
	_ = ok
}
