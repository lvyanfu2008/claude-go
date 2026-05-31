package store

import (
	"sync"
	"testing"
)

func TestAtomGetSet(t *testing.T) {
	atom := NewAtom(42)
	if atom.Get() != 42 {
		t.Fatalf("expected 42, got %d", atom.Get())
	}
	atom.Set(99)
	if atom.Get() != 99 {
		t.Fatalf("expected 99, got %d", atom.Get())
	}
}

func TestAtomWatch(t *testing.T) {
	atom := NewAtom("hello")
	var mu sync.Mutex
	var received []string
	unsub := atom.Watch(func(v string) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	atom.Set("world")
	atom.Set("foo")
	unsub()
	atom.Set("bar")
	mu.Lock()
	if len(received) != 2 {
		t.Fatalf("expected 2 notifications, got %d: %v", len(received), received)
	}
	if received[0] != "world" || received[1] != "foo" {
		t.Fatalf("unexpected values: %v", received)
	}
	mu.Unlock()
}

func TestAtomWatchMultipleSubscribers(t *testing.T) {
	atom := NewAtom(0)
	count := 0
	var mu sync.Mutex
	inc := func(v int) { mu.Lock(); count++; mu.Unlock() }
	atom.Watch(inc)
	atom.Watch(inc)
	atom.Set(1)
	mu.Lock()
	if count != 2 {
		t.Fatalf("expected 2 increments, got %d", count)
	}
	mu.Unlock()
}

func TestAtomVersion(t *testing.T) {
	atom := NewAtom(0)
	v1 := atom.Version()
	atom.Set(1)
	v2 := atom.Version()
	if v2 <= v1 {
		t.Fatal("version should increment after Set")
	}
}
