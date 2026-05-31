package store

import (
	"sync"
	"sync/atomic"
)

type AtomReader interface {
	Version() uint64
}

type Atom[T any] struct {
	mu       sync.RWMutex
	value    T
	version  uint64
	watchers map[uint64]func(T)
	nextID   uint64
}

func NewAtom[T any](initial T) *Atom[T] {
	return &Atom[T]{
		value:    initial,
		watchers: make(map[uint64]func(T)),
	}
}

func (a *Atom[T]) Get() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.value
}

func (a *Atom[T]) Set(val T) {
	a.mu.Lock()
	a.value = val
	atomic.AddUint64(&a.version, 1)
	watchers := make([]func(T), 0, len(a.watchers))
	for _, w := range a.watchers {
		watchers = append(watchers, w)
	}
	a.mu.Unlock()
	for _, w := range watchers {
		w(val)
	}
}

func (a *Atom[T]) Watch(fn func(T)) func() {
	a.mu.Lock()
	id := a.nextID
	a.nextID++
	a.watchers[id] = fn
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		delete(a.watchers, id)
		a.mu.Unlock()
	}
}

func (a *Atom[T]) Version() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.version
}

// SetInterface provides type-erased access to Set. It is used by Transaction.Set
// via the AtomSetter interface.
func (a *Atom[T]) SetInterface(val interface{}) {
	if v, ok := val.(T); ok {
		a.Set(v)
	}
}
