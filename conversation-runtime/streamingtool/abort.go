package streamingtool

import "sync"

// AbortController mirrors the DOM AbortController used in StreamingToolExecutor.ts
// (createChildAbortController in src/utils/abortController.ts: child aborts when parent aborts;
// aborting the child does not abort the parent).
type AbortController struct {
	sig *AbortSignal
}

// AbortSignal mirrors AbortSignal (aborted + reason + propagation).
type AbortSignal struct {
	mu        sync.Mutex
	aborted   bool
	reason    any
	listeners []func()
}

// NewAbortController creates a root controller (TS: createAbortController).
func NewAbortController() *AbortController {
	return &AbortController{sig: &AbortSignal{}}
}

// Signal returns the signal for this controller.
func (a *AbortController) Signal() *AbortSignal { return a.sig }

// Aborted reports whether Abort was called.
func (s *AbortSignal) Aborted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.aborted
}

// Reason returns the reason passed to Abort (nil if not aborted).
func (s *AbortSignal) Reason() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reason
}

// Abort mirrors AbortController.abort(reason).
func (a *AbortController) Abort(reason any) {
	if a == nil || a.sig == nil {
		return
	}
	a.sig.abort(reason)
}

func (s *AbortSignal) abort(reason any) {
	s.mu.Lock()
	if s.aborted {
		s.mu.Unlock()
		return
	}
	s.aborted = true
	s.reason = reason
	ls := s.listeners
	s.listeners = nil
	s.mu.Unlock()
	for _, f := range ls {
		f()
	}
}

func (s *AbortSignal) addListener(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aborted {
		go f()
		return
	}
	s.listeners = append(s.listeners, f)
}

// AddListener mirrors signal addEventListener (no 'once'; wrap caller-side if needed).
func (s *AbortSignal) AddListener(f func()) { s.addListener(f) }

// OnAbortOnce runs cb exactly once when this controller is aborted (TS once: true listener).
func (a *AbortController) OnAbortOnce(cb func(reason any)) {
	if a == nil || a.sig == nil {
		return
	}
	var once sync.Once
	a.sig.AddListener(func() {
		once.Do(func() { cb(a.sig.Reason()) })
	})
}

// CreateChildAbortController mirrors createChildAbortController(parent) from src/utils/abortController.ts.
func CreateChildAbortController(parent *AbortController) *AbortController {
	child := NewAbortController()
	if parent == nil || parent.sig == nil {
		return child
	}
	if parent.sig.Aborted() {
		child.Abort(parent.sig.Reason())
		return child
	}
	parent.sig.addListener(func() {
		child.Abort(parent.sig.Reason())
	})
	return child
}
