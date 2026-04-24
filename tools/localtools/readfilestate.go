package localtools

import (
	"sync"
)

// ReadFileEntry mirrors src/Tool.ts readFileState map values used by FileReadTool / FileWriteTool / FileEditTool.
type ReadFileEntry struct {
	Content        string
	Timestamp      int64 // floor(mtime ms), like TS Math.floor(mtimeMs)
	Offset         *int  // nil when TS stores undefined (e.g. after Write); set by Read text path
	Limit          *int
	IsPartialView  bool
}

// IsFullReadEntry mirrors TS: readTimestamp.offset === undefined && readTimestamp.limit === undefined
// (Write/Edit staleness uses this for content fallback when mtime changes without content change).
func IsFullReadEntry(e *ReadFileEntry) bool {
	return e != nil && e.Offset == nil && e.Limit == nil
}

// ReadFileState is a session-scoped map path -> last read snapshot (TS toolUseContext.readFileState).
type ReadFileState struct {
	mu sync.Mutex
	m  map[string]*ReadFileEntry
}

func NewReadFileState() *ReadFileState {
	return &ReadFileState{m: make(map[string]*ReadFileEntry)}
}

func (s *ReadFileState) Get(absPath string) *ReadFileEntry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[absPath]
}

func (s *ReadFileState) Set(absPath string, e *ReadFileEntry) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[absPath] = e
}

// Keys returns all file paths currently tracked in the state (mirrors TS FileStateCache.keys()).
// The returned slice is a snapshot copy.
func (s *ReadFileState) Keys() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}
