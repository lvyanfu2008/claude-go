package store

import (
	"sync"
	"time"

	"goc/gou/ink/vdom"
)

type Store struct {
	mu       sync.RWMutex
	atoms    map[string]interface{}
	renderCh chan struct{}
	onRender func()
	closeCh  chan struct{}

	// Legacy compat fields for StoreReader interface.
	// These mirror the atom values so components using StoreReader can read state.
	messages       []vdom.Message
	streamingText  string
	streamingTools []vdom.StreamingToolUse
	inputValue     string
	cursorPos      int
	isLoading      bool
	width, height  int
	scrollTop      int
	meta           map[string]string
}

// StoreReader-compatible methods. These are the legacy read path for
// components that haven't been migrated to use atoms directly.

func (s *Store) GetMessages() []vdom.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messages
}

func (s *Store) StreamingText() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamingText
}

func (s *Store) StreamingTools() []vdom.StreamingToolUse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamingTools
}

func (s *Store) InputValue() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inputValue
}

func (s *Store) CursorPos() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cursorPos
}

func (s *Store) IsLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLoading
}

func (s *Store) Width() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.width
}

func (s *Store) Height() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.height
}

func (s *Store) GetMeta(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta[key]
}

// Setters for legacy compat fields.

func (s *Store) SetMessages(v []vdom.Message) {
	s.mu.Lock()
	s.messages = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetStreamingText(v string) {
	s.mu.Lock()
	s.streamingText = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetStreamingTools(v []vdom.StreamingToolUse) {
	s.mu.Lock()
	s.streamingTools = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetInputValue(v string) {
	s.mu.Lock()
	s.inputValue = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetCursorPos(v int) {
	s.mu.Lock()
	s.cursorPos = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetLoading(v bool) {
	s.mu.Lock()
	s.isLoading = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetWidth(v int) {
	s.mu.Lock()
	s.width = v
	s.mu.Unlock()
}

func (s *Store) SetHeight(v int) {
	s.mu.Lock()
	s.height = v
	s.mu.Unlock()
}

func (s *Store) ScrollTop() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scrollTop
}

func (s *Store) SetScrollTop(v int) {
	s.mu.Lock()
	s.scrollTop = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetMeta(key, val string) {
	s.mu.Lock()
	if s.meta == nil {
		s.meta = make(map[string]string)
	}
	s.meta[key] = val
	s.mu.Unlock()
}

func NewStore() *Store {
	return &Store{
		atoms:    make(map[string]interface{}),
		renderCh: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
	}
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
