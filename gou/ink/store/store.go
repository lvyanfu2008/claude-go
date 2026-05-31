package store

import (
	"sync"
	"time"
)

type Store struct {
	mu       sync.RWMutex
	atoms    map[string]interface{}
	renderCh chan struct{}
	onRender func()
	closeCh  chan struct{}
}

func NewStore() *Store {
	s := &Store{
		atoms:    make(map[string]interface{}),
		renderCh: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
	}
	go s.RunRenderLoop()
	return s
}

func DefineAtom[T any](s *Store, key string, initial T) *Atom[T] {
	a := NewAtom(initial)
	s.mu.Lock()
	s.atoms[key] = a
	s.mu.Unlock()
	return a
}

func (s *Store) SetOnRender(fn func()) {
	s.onRender = fn
}

// ScheduleRender enqueues a render signal. Multiple rapid calls are coalesced
// into a single render via the buffered channel (capacity 1).
func (s *Store) ScheduleRender() {
	select {
	case s.renderCh <- struct{}{}:
	default:
	}
}

// RunRenderLoop processes render signals. It must be called from a goroutine.
func (s *Store) RunRenderLoop() {
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.renderCh:
			if s.onRender != nil {
				s.onRender()
			}
		case <-ticker.C:
		case <-s.closeCh:
			return
		}
	}
}

func (s *Store) Stop() {
	close(s.closeCh)
}

// Batch executes fn inside a transaction. All atom mutations within the batch
// are followed by a single coalesced render signal.
func (s *Store) Batch(fn func(tx *Transaction)) {
	tx := &Transaction{store: s}
	fn(tx)
	tx.done()
}

// AtomSetter is the interface implemented by Atom[T] for type-erased access.
type AtomSetter interface {
	SetInterface(val interface{})
}

type Transaction struct {
	store      *Store
	committed  bool
}

func (tx *Transaction) Set(atomRaw interface{}, val interface{}) {
	if as, ok := atomRaw.(AtomSetter); ok {
		as.SetInterface(val)
	}
}

func (tx *Transaction) done() {
	if tx.committed {
		return
	}
	tx.committed = true
	tx.store.ScheduleRender()
}
