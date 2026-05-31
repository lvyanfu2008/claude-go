package store

import (
	"testing"
	"time"
)

func TestStoreDefineAtom(t *testing.T) {
	s := NewStore()
	a := DefineAtom(s, "messages", []string{})
	_ = a
}

func TestStoreBatchCommit(t *testing.T) {
	s := NewStore()
	go s.RunRenderLoop()
	defer s.Stop()

	a := DefineAtom(s, "count", 0)
	b := DefineAtom(s, "text", "")

	renders := 0
	s.SetOnRender(func() { renders++ })

	s.Batch(func(tx *Transaction) {
		tx.Set(a, 10)
		tx.Set(b, "hello")
	})

	if a.Get() != 10 {
		t.Fatalf("expected 10, got %d", a.Get())
	}
	if b.Get() != "hello" {
		t.Fatalf("expected hello, got %q", b.Get())
	}
	time.Sleep(20 * time.Millisecond)
	if renders != 1 {
		t.Fatalf("expected 1 render after batch, got %d", renders)
	}
}

func TestStoreDefineAtomTyped(t *testing.T) {
	s := NewStore()
	a := DefineAtom(s, "name", "default")
	if a.Get() != "default" {
		t.Fatalf("expected default, got %q", a.Get())
	}
	a.Set("alice")
	if a.Get() != "alice" {
		t.Fatalf("expected alice, got %q", a.Get())
	}
}

func TestStoreRenderCoalesce(t *testing.T) {
	s := NewStore()
	go s.RunRenderLoop()
	defer s.Stop()

	renders := 0
	s.SetOnRender(func() { renders++ })

	for i := 0; i < 5; i++ {
		s.ScheduleRender()
	}
	time.Sleep(30 * time.Millisecond)
	if renders > 2 {
		t.Fatalf("expected 1-2 renders (coalesced), got %d", renders)
	}
}
