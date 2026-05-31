package store

import "sync"

type Selector struct {
	deps      []AtomReader
	compute   func() interface{}
	cached    interface{}
	cachedVer uint64
	mu        sync.RWMutex
}

func NewSelector(deps []AtomReader, compute func() interface{}) *Selector {
	return &Selector{deps: deps, compute: compute, cachedVer: ^uint64(0)}
}

func (s *Selector) Get() interface{} {
	ver := s.combinedVersion()
	s.mu.RLock()
	if ver == s.cachedVer {
		v := s.cached
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// double-check after acquiring write lock
	ver = s.combinedVersion()
	if ver == s.cachedVer {
		return s.cached
	}
	s.cached = s.compute()
	s.cachedVer = ver
	return s.cached
}

func (s *Selector) combinedVersion() uint64 {
	var v uint64
	for _, d := range s.deps {
		v += d.Version()
	}
	return v
}

func (s *Selector) Version() uint64 {
	return s.combinedVersion()
}

// TypedSelector wraps Selector with type safety.
type TypedSelector[T any] struct {
	raw *Selector
}

func NewTypedSelector[T any](deps []AtomReader, compute func() T) *TypedSelector[T] {
	return &TypedSelector[T]{
		raw: NewSelector(deps, func() interface{} { return compute() }),
	}
}

func (ts *TypedSelector[T]) Get() T {
	return ts.raw.Get().(T)
}

func (ts *TypedSelector[T]) Version() uint64 {
	return ts.raw.Version()
}
