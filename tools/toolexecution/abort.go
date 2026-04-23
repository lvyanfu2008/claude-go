package toolexecution

import "sync"

// AbortController mirrors DOM AbortController used by toolExecution.ts / ToolUseContext.abortController.
type AbortController struct {
	sig *AbortSignal
}

// AbortSignal mirrors AbortSignal.aborted / reason.
type AbortSignal struct {
	mu      sync.Mutex
	aborted bool
	reason  any
}

func NewAbortController() *AbortController {
	return &AbortController{sig: &AbortSignal{}}
}

func (a *AbortController) Signal() *AbortSignal {
	if a == nil {
		return nil
	}
	return a.sig
}

func (s *AbortSignal) Aborted() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.aborted
}

func (s *AbortSignal) Reason() any {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reason
}

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
	s.mu.Unlock()
}
