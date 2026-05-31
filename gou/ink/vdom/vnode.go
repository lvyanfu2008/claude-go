package vdom

import "goc/gou/theme"

// StoreReader is the read-only interface that the VDOM layer uses to read
// application state from the store without depending on the store package.
type StoreReader interface {
	GetMessages() []Message
	StreamingText() string
	StreamingTools() []StreamingToolUse
	InputValue() string
	CursorPos() int
	IsLoading() bool
	Width() int
	Height() int
	GetMeta(key string) string
}

// Message represents a single message in the conversation.
type Message struct {
	UUID          string
	Type          string
	ContentBlocks []ContentBlock
	Meta          map[string]interface{}
}

// ContentBlock represents a content block within a message.
type ContentBlock struct {
	Type    string
	Content string
	Name    string
	Input   map[string]interface{}
	State   string
	Result  *ContentBlock
	Meta    map[string]interface{}
}

// StreamingToolUse represents a tool invocation during streaming.
type StreamingToolUse struct {
	UUID  string
	Name  string
	Input map[string]interface{}
}

type VNode struct {
	Type     string
	Key      string
	Props    Props
	Children []VNode
	Layout   LayoutResult
}

type Props map[string]interface{}

func (p Props) GetString(key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p Props) GetInt(key string) int {
	if v, ok := p[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func (p Props) GetBool(key string) bool {
	if v, ok := p[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (p Props) Get(key string) interface{} {
	return p[key]
}

type LayoutResult struct {
	X, Y         int
	W, H         int
	ContentH     int
	OverflowTop  int
	VisibleRange [2]int
}

type Constraints struct {
	MinW, MaxW int
	MinH, MaxH int
}

func Unbounded() Constraints {
	return Constraints{MinW: 0, MaxW: 1<<31 - 1, MinH: 0, MaxH: 1<<31 - 1}
}

type Context struct {
	Theme    *theme.Palette
	Store    StoreReader
	Schedule func()

	// Internal: current fiber during reconciliation
	fiber     *Fiber
	hookIndex int
}

// HookState represents the state of a single hook within a component fiber.
type HookState struct {
	state      interface{}
	deps       []interface{}
	cleanup    func()
	memoized   interface{}
	effectRun  bool
}

// Fiber is the internal representation of a VNode during reconciliation.
type Fiber struct {
	vnode       *VNode
	child       *Fiber
	sibling     *Fiber
	returnFiber *Fiber
	effectTag   EffectTag
	hooks       []HookState
	deleted     bool
}

type Component func(ctx *Context, props Props) VNode
