package ink

import (
	"testing"
	"time"
)

func TestStoreScheduleRender_coalescing(t *testing.T) {
	s := NewStore()
	count := 0
	s.SetOnRender(func() { count++ })
	for i := 0; i < 10; i++ {
		s.ScheduleRender()
	}
	select {
	case <-s.renderCh:
		s.onRender()
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected render signal")
	}
	if count != 1 {
		t.Errorf("expected 1 render (coalesced), got %d", count)
	}
}

func TestStoreAppendMessage(t *testing.T) {
	s := NewStore()
	rendered := false
	s.SetOnRender(func() { rendered = true })
	s.AppendMessage(Message{UUID: "1", Type: "user"})
	// Drain the channel to trigger the render callback
	select {
	case <-s.renderCh:
		if s.onRender != nil {
			s.onRender()
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected render signal")
	}
	if !rendered {
		t.Error("append should trigger render")
	}
}
