package appstate

import "sync"

// Store is a minimal thread-safe holder for [AppState], analogous to src/state/store.ts Store<AppState>
// and headless bootstrapContext getAppState / setAppState.
type Store struct {
	mu sync.RWMutex
	s  AppState
}

// NewStore returns a store with a copy of initial (caller may use [DefaultAppState]).
func NewStore(initial AppState) *Store {
	return &Store{s: initial}
}

// GetState returns the current snapshot (TS store.getState).
func (st *Store) GetState() AppState {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s
}

// SetState replaces state (TS store.setState(next) when next is not a function).
func (st *Store) SetState(next AppState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s = next
}

// Update applies fn to the current state (TS setState(prev => next)).
func (st *Store) Update(fn func(prev AppState) AppState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s = fn(st.s)
}
