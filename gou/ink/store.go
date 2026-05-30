package ink

import (
	"sync"
	"time"
)

type Store struct {
	mu sync.RWMutex

	Messages       []Message
	StreamingText  string
	StreamingTools []StreamingToolUse
	ScrollTop      int
	InputValue     string
	IsLoading      bool
	Width, Height  int

	renderCh chan struct{}
	onRender func()
	closeCh  chan struct{}
}

type Message struct {
	UUID          string
	Type          string
	ContentBlocks []ContentBlock
	Meta          map[string]interface{}
}

type ContentBlock struct {
	Type    string
	Content string
	Name    string
	Input   map[string]interface{}
	State   string
	Result  *ContentBlock
	Meta    map[string]interface{}
}

type StreamingToolUse struct {
	UUID  string
	Name  string
	Input map[string]interface{}
}

func NewStore() *Store {
	return &Store{
		renderCh: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
	}
}

func (s *Store) SetOnRender(fn func()) { s.onRender = fn }

func (s *Store) ScheduleRender() {
	select {
	case s.renderCh <- struct{}{}:
	default:
	}
}

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

func (s *Store) Stop() { close(s.closeCh) }

func (s *Store) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Messages
}

func (s *Store) AppendMessage(msg Message) {
	s.mu.Lock()
	s.Messages = append(s.Messages, msg)
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetStreamingText(text string) {
	s.mu.Lock()
	s.StreamingText = text
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetStreamingTools(tools []StreamingToolUse) {
	s.mu.Lock()
	s.StreamingTools = tools
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetLoading(v bool) {
	s.mu.Lock()
	s.IsLoading = v
	s.mu.Unlock()
	s.ScheduleRender()
}

func (s *Store) SetInputValue(v string) {
	s.mu.Lock()
	s.InputValue = v
	s.mu.Unlock()
	s.ScheduleRender()
}
