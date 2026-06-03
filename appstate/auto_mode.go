package appstate

import (
	"sync"
)

// AutoModeState tracks whether auto mode is currently active (classifier-driven).
// Mirrors TS autoModeStateModule singleton.
type AutoModeState struct {
	mu       sync.Mutex
	isActive bool
}

// IsActive returns true if auto mode is currently active.
func (s *AutoModeState) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActive
}

// SetActive sets the auto mode active flag.
func (s *AutoModeState) SetActive(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isActive = v
}

// GlobalAutoModeState is the process-wide auto mode state singleton.
var GlobalAutoModeState = &AutoModeState{}
