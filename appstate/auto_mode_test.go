package appstate

import (
	"sync"
	"testing"
)

func TestAutoModeState_Concurrent(t *testing.T) {
	s := &AutoModeState{}
	var wg sync.WaitGroup
	n := 100

	// Concurrent writes
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(v bool) {
			defer wg.Done()
			s.SetActive(v)
		}(i%2 == 0)
	}

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.IsActive()
		}()
	}

	wg.Wait()
	// No race detector errors = pass
}

func TestAutoModeState_BasicToggle(t *testing.T) {
	s := &AutoModeState{}
	if s.IsActive() {
		t.Error("should be inactive by default")
	}
	s.SetActive(true)
	if !s.IsActive() {
		t.Error("should be active after SetActive(true)")
	}
	s.SetActive(false)
	if s.IsActive() {
		t.Error("should be inactive after SetActive(false)")
	}
}
