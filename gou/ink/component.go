package ink

import (
	"goc/gou/ink/core"
	"goc/gou/ink/layout"
	"goc/gou/ink/render"
	"goc/gou/ink/store"
	"goc/gou/ink/vdom"
)

// ---- vdom re-exports ----
type (
	VNode            = vdom.VNode
	Props            = vdom.Props
	Constraints      = vdom.Constraints
	LayoutResult     = vdom.LayoutResult
	Component        = vdom.Component
	Context          = vdom.Context
	Message          = vdom.Message
	ContentBlock     = vdom.ContentBlock
	StreamingToolUse = vdom.StreamingToolUse
	StoreReader      = vdom.StoreReader
	Fiber            = vdom.Fiber
	HookState        = vdom.HookState
	EffectTag        = vdom.EffectTag
)

// ---- render re-exports ----
type (
	Screen     = render.Screen
	TermCell   = render.TermCell
	CellStyle  = render.CellStyle
	DiffEngine = render.DiffEngine
)

// ---- layout re-exports ----
type (
	VirtualScrollState = layout.VirtualScrollState
)

// ---- store re-exports ----
type (
	AtomReader           = store.AtomReader
	Atom[T any]          = store.Atom[T]
	Selector             = store.Selector
	TypedSelector[T any] = store.TypedSelector[T]
	Store                = store.Store
	Transaction          = store.Transaction
)

// ---- core re-exports ----
type (
	ParsedKey      = core.ParsedKey
	MouseEvent     = core.MouseEvent
	MouseEventType = core.MouseEventType
	Modifier       = core.Modifier
)

// ---- vdom constants ----
const (
	NoEffect    = vdom.NoEffect
	Placement   = vdom.Placement
	Update      = vdom.Update
	Deletion    = vdom.Deletion
	Replacement = vdom.Replacement
)

// ---- core constants ----
const (
	Ctrl  = core.Ctrl
	Alt   = core.Alt
	Shift = core.Shift
	Meta  = core.Meta

	MousePress   = core.MousePress
	MouseRelease = core.MouseRelease
	MouseMove    = core.MouseMove
	MouseWheel   = core.MouseWheel
)

// ---- non-generic function re-exports ----
var (
	NewTerminal          = core.NewTerminal
	NewKeyboardParser    = core.NewKeyboardParser
	IsMouseEvent         = core.IsMouseEvent
	DecodeMouse          = core.DecodeMouse
	IsBracketedPasteStart = core.IsBracketedPasteStart
	IsBracketedPasteEnd  = core.IsBracketedPasteEnd

	NewScreen            = render.NewScreen
	NewDiffEngine        = render.NewDiffEngine
	Rasterize            = render.Rasterize
	StyleToSGR           = render.StyleToSGR
	SgrReset             = render.SgrReset
	CursorTo             = render.CursorTo
	CursorUp             = render.CursorUp
	CursorDown           = render.CursorDown
	EraseToEnd           = render.EraseToEnd
	EraseLine            = render.EraseLine
	EraseDisplay         = render.EraseDisplay

	ComputeLayout        = layout.ComputeLayout
	NewVirtualScrollState = layout.NewVirtualScrollState

	NewStore             = store.NewStore
	NewSelector          = store.NewSelector
	Unbounded            = vdom.Unbounded
)

// ---- generic function wrappers ----

// NewAtom creates a new reactive Atom with the given initial value.
func NewAtom[T any](initial T) *Atom[T] {
	return store.NewAtom[T](initial)
}

// DefineAtom defines a named atom in the store with the given initial value.
func DefineAtom[T any](s *Store, key string, initial T) *Atom[T] {
	return store.DefineAtom[T](s, key, initial)
}

// NewTypedSelector creates a type-safe memoized Selector over the given atom deps.
func NewTypedSelector[T any](deps []AtomReader, compute func() T) *TypedSelector[T] {
	return store.NewTypedSelector[T](deps, compute)
}
